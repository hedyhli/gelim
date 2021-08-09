package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-xdg"
	"github.com/manifoldco/ansiwrap"
	ln "github.com/peterh/liner"
	"golang.org/x/term"
)

type Page struct {
	bodyReader io.Reader
	body       []gemini.Line
	mediaType  string
	params     map[string]string
	u          *url.URL
}

type Client struct {
	links            []string
	inputLinks       []int // contains index to links in `links` that needs spartan input
	history          []*url.URL
	conf             *Config
	mainReader       *ln.State
	inputReader      *ln.State
	promptHistory    *os.File
	inputHistory     *os.File
	promptSuggestion string
	gmi              *gemini.Client

	// page             *Page
	pre         bool
	tmpPageFile *os.File
	currentURL  *url.URL
}

func NewClient() (*Client, error) {
	var c Client
	var err error
	// load config
	conf, err := LoadConfig()
	if err != nil {
		return &c, err
	}
	// c.history = make([]*url.URL, 100)
	c.links = make([]string, 100)
	c.conf = conf
	c.gmi = &gemini.Client{}
	c.mainReader = ln.NewLiner()
	c.mainReader.SetCtrlCAborts(true)
	c.inputReader = ln.NewLiner()
	c.inputReader.SetCtrlCAborts(true)

	c.tmpPageFile, err = os.CreateTemp("", "*")
	if err != nil {
		return &c, err
	}

	dataDir := filepath.Join(xdg.DataHome(), "gelim")

	// Create cache/data/runtime dirs/files
	os.MkdirAll(dataDir, 0700)
	c.promptHistory, err = os.OpenFile(filepath.Join(dataDir, "prompt_history.txt"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return &c, err
	}
	c.inputHistory, err = os.OpenFile(filepath.Join(dataDir, "input_history.txt"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return &c, err
	}
	c.mainReader.ReadHistory(c.promptHistory)
	c.inputReader.ReadHistory(c.inputHistory)

	c.mainReader.SetCompleter(CommandCompleter)
	return &c, err
}

func (c *Client) QuitClient() {
	c.mainReader.WriteHistory(c.promptHistory)
	c.inputReader.WriteHistory(c.inputHistory)
	c.promptHistory.Close()
	c.inputHistory.Close()
	c.mainReader.Close()
	c.inputReader.Close()
	os.Remove(c.tmpPageFile.Name())
}

func (c *Client) GetLinkFromIndex(i int) (link string, spartanInput bool) {
	spartanInput = false
	if len(c.links) < i {
		fmt.Println(ErrorColor("invalid link index, I have %d links so far", len(c.links)))
		return
	}
	link = c.links[i-1]
	for _, v := range c.inputLinks {
		if i-1 == v {
			spartanInput = true
			return
		}
	}
	return
}

func (c *Client) DisplayPage(page *Page) {
	c.tmpPageFile.Truncate(0)
	c.tmpPageFile.Seek(0, 0)
	// text/* content only for now
	// TODO: support more media types
	if !strings.HasPrefix(page.mediaType, "text/") && page.mediaType != "application/octet-stream" {
		fmt.Println(ErrorColor("Unsupported type " + page.mediaType))
		return
	}
	if page.mediaType == "text/gemini" {
		c.pre = false
		err := gemini.ParseLines(page.bodyReader, c.GemtextHandler)
		if err != nil {
			fmt.Println(err.Error())
		}
		Pager(c.tmpPageFile.Name(), c.conf)
		return
	}
	if page.mediaType == "application/octet-stream" || strings.HasPrefix(page.mediaType, "text/") {
		scanner := bufio.NewScanner(page.bodyReader)
		for scanner.Scan() {
			c.AppendLine(scanner.Text())
		}
		Pager(c.tmpPageFile.Name(), c.conf)
		return
	}
}

func (c *Client) AppendLine(line string) {
	c.tmpPageFile.WriteString(line + "\n")
}

func (c *Client) GemtextHandler(line gemini.Line) {
	current := c.currentURL
	width, _, err := term.GetSize(0)
	if err != nil {
		fmt.Println(ErrorColor("error getting terminal size"))
		return
	}
	switch line := line.(type) {
	case gemini.LineHeading1:
		c.AppendLine(ansiwrap.Wrap(h1Style(line), width))
	case gemini.LineHeading2:
		c.AppendLine(ansiwrap.Wrap(h2Style(line), width))
	case gemini.LineListItem:
		// Using width - 3 because of 3 spaces "   " indent at the start
		c.AppendLine("  • " + ansiwrap.WrapIndent(string(line), width-3, 0, 4))
	case gemini.LinePreformattedText:
		c.AppendLine(string(line))
	case gemini.LinePreformattingToggle:
		c.pre = !c.pre // TODO: idk what to do here
	case gemini.LineQuote:
		// appending extra \n here because we want quote blocks to stand out
		// with leading and trailing new lines to distinguish from paragraphs
		// as well as making it clear that it's actually a quote block.
		// NOT doing this anymore!
		// (because it looked bad if quotes are continuous)
		// TODO: remove extra new lines in the end
		c.AppendLine(" > " + ansiwrap.WrapIndent(string(line), width, 1, 4))
	case gemini.LineLink:
		parsedLink, err := url.Parse(line.URL)
		if err != nil {
			return
		}
		link := current.ResolveReference(parsedLink) // link url
		label := line.Name
		if label == "" {
			label = line.URL
		}
		c.links = append(c.links, link.String())
		linkLine := fmt.Sprintf("[%d] ", len(c.links)) + linkStyle(label)
		if link.Scheme != "gemini" {
			linkLine += fmt.Sprintf(" (%s)", link.Scheme)
		}
		c.AppendLine(ansiwrap.Wrap(linkLine, width))
	case gemini.LineText:
		if strings.HasPrefix(string(line), "=:") {
			text := string(line)[2:]
			text = strings.TrimLeft(text, " \t")
			split := strings.IndexAny(text, " \t")
			var (
				u    string
				name string
			)
			if split == -1 {
				// text is a URL
				u = text
				name = u
			} else {
				u = text[:split]
				name = text[split:]
				name = strings.TrimLeft(name, " \t")
			}
			label := name + " [INPUT]"
			c.inputLinks = append(c.inputLinks, len(c.links)) // using len(c.links) because it is only appended below so the value from that is just right
			parsedLink, err := url.Parse(u)
			if err != nil {
				return // TODO: do something better than ignoring the link
			}
			link := current.ResolveReference(parsedLink) // link url
			c.links = append(c.links, link.String())
			linkLine := fmt.Sprintf("[%d] ", len(c.links)) + linkStyle(label)
			if link.Scheme != "gemini" {
				linkLine += fmt.Sprintf(" (%s)", link.Scheme)
			}
			c.AppendLine(ansiwrap.Wrap(linkLine, width))
		} else {
			c.AppendLine(ansiwrap.Wrap(string(line), width))
		}
	}
	// Remove last \n
	// if len(c.rendered) > 0 {
	// 	c.rendered = c.rendered[:len(c.rendered)-1]
	// }
}

// Input handles Input status codes
func (c *Client) Input(u string, sensitive bool) (ok bool) {
	var query string
	var err error
	// c.inputReader.SetMultiLineMode(true)
	if sensitive {
		query, err = c.inputReader.PasswordPrompt("INPUT (sensitive)> ")
	} else {
		query, err = c.inputReader.Prompt("INPUT> ")
	}
	if err != nil {
		if err == ln.ErrPromptAborted {
			fmt.Println(ErrorColor("\ninput cancelled"))
			return false
		}
		fmt.Println(ErrorColor("\nerror reading input:"))
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	if !sensitive {
		c.inputReader.AppendHistory(query)
	}
	u = u + "?" + queryEscape(query)
	return c.HandleURL(u)
}

func (c *Client) HandleURL(u string) bool {
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		fmt.Println(ErrorColor("invalid url"))
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		// have to parse again
		parsed, err = url.Parse("gemini://" + u)
		if err != nil {
			fmt.Println(ErrorColor("invalid url"))
			return false
		}
	}
	return c.HandleParsedURL(parsed)
}

// Handles either a spartan URL or a gemini URL
func (c *Client) HandleParsedURL(parsed *url.URL) bool {
	// TODO; config proxies or program to do other shemes
	if parsed.Scheme != "gemini" && parsed.Scheme != "spartan" {
		fmt.Println(ErrorColor("Unsupported scheme %s", parsed.Scheme))
		return false
	}
	if parsed.Scheme == "gemini" {
		return c.HandleGeminiParsedURL(parsed)
	}
	return c.HandleSpartanParsedURL(parsed)
}

func (c *Client) HandleSpartanParsedURL(parsed *url.URL) bool {
	c.currentURL = parsed
	res, err := SpartanParsedURL(parsed)
	if err != nil {
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	defer (*res.conn).Close()
	c.links = make([]string, 0, 100) // reset links
	c.inputLinks = make([]int, 0, 100)

	page := &Page{u: parsed, bodyReader: res.bodyReader}
	// Handle status
	switch res.status {
	case 2:
		mediaType, params, err := ParseMeta(res.meta)
		if err != nil {
			fmt.Println(ErrorColor("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		c.HandleURL("spartan://" + parsed.Host + res.meta)
	case 4:
		fmt.Println("Error: " + res.meta)
	case 5:
		fmt.Println("Server error: " + res.meta)
	}

	if (len(c.history) > 0) && (c.history[len(c.history)-1].String() != parsed.String()) || len(c.history) == 0 {
		c.history = append(c.history, parsed)
	}
	return true
}

func (c *Client) HandleGeminiParsedURL(parsed *url.URL) bool {
	c.currentURL = parsed
	ctx := context.Background()
	res, err := c.gmi.Get(ctx, parsed.String())
	if err != nil {
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	conn := res.Conn()
	defer conn.Close()
	c.links = make([]string, 0, 100) // reset links
	c.inputLinks = make([]int, 0, 100)

	// mediaType and params will be parsed later
	page := &Page{u: parsed, bodyReader: res.Body}
	statusGroup := res.Status / 10 // floor division
	switch statusGroup {
	case 1:
		u := strings.TrimRight(page.u.String(), "?"+page.u.RawQuery)
		fmt.Println(res.Meta)
		if res.Status == 11 {
			return c.Input(u, true) // sensitive input
		}
		return c.Input(u, false)
	case 2:
		mediaType, params, err := ParseMeta(res.Meta)
		if err != nil {
			fmt.Println(ErrorColor("Unable to parse header meta\"%s\": %s", res.Meta, err))
			return false
		}
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		return c.HandleURL(res.Meta) // TODO: max redirect times
	case 4, 5:
		switch res.Status {
		case 40:
			fmt.Println(ErrorColor("Temperorary failure"))
		case 41:
			fmt.Println(ErrorColor("Server unavailable"))
		case 42:
			fmt.Println(ErrorColor("CGI error"))
		case 43:
			fmt.Println(ErrorColor("Proxy error"))
		case 44:
			fmt.Println(ErrorColor("Slow down"))
		case 52:
			fmt.Println(ErrorColor("Gone"))
		}
		fmt.Println(ErrorColor("%d %s", res.Status, res.Meta))
	case 6:
		fmt.Println(res.Meta)
		fmt.Println("Sorry, gelim does not support client certificates yet.")
	default:
		fmt.Println(ErrorColor("invalid status code %d", res.Status))
		return false
	}
	if (len(c.history) > 0) && (c.history[len(c.history)-1].String() != parsed.String()) || len(c.history) == 0 {
		c.history = append(c.history, parsed)
	}
	return true
}

func (c *Client) Search(query string) {
	u := c.conf.SearchURL + "?" + queryEscape(query)
	c.HandleURL(u)
}

func (c *Client) LookupCommand(cmdStr string) (cmd Command, ok bool) {
	cmdName := ""
	ok = false
	// skipping metaCommands
	for name, v := range commands {
		if name == cmdStr {
			cmdName = name
			break
		}
		for _, alias := range v.aliases {
			if alias == cmdStr {
				cmdName = name
				break
			}
		}
	}
	if cmdName == "" {
		return
	}
	cmd = commands[cmdName]
	ok = true
	return
}

func (c *Client) Command(cmdStr string, args ...string) bool {
	cmdName := ""
	for name, v := range metaCommands {
		if name == cmdStr {
			cmdName = name
			break
		}
		for _, alias := range v.aliases {
			if alias == cmdStr {
				cmdName = name
				break
			}
		}
	}
	if cmdName != "" {
		metaCommands[cmdName].do(c, args...)
		return true
	}
	cmd, ok := c.LookupCommand(cmdStr)
	if !ok {
		return ok
	}
	cmd.do(c, args...)
	return true
}
