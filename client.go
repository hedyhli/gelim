package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~adnano/go-xdg"
	"github.com/manifoldco/ansiwrap"
	ln "github.com/peterh/liner"
	"golang.org/x/term"
)

type Page struct {
	bodyBytes []byte
	mediaType string
	params    map[string]string
	u         *url.URL
}

type Client struct {
	links         []string
	inputLinks    []int // contains index to links in `links` that needs spartan input
	history       []*url.URL
	conf          *Config
	mainReader    *ln.State
	inputReader   *ln.State
	promptHistory *os.File
	inputHistory  *os.File
	promptSuggestion string
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
	c.mainReader = ln.NewLiner()
	c.mainReader.SetCtrlCAborts(true)
	c.inputReader = ln.NewLiner()
	c.inputReader.SetCtrlCAborts(true)

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
	os.Exit(0)
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
	// TODO: proper stream - read the reader and stuff
	if page.mediaType == "application/octet-stream" {
		Pager(string(page.bodyBytes), c.conf)
		return
	}
	// text/* content only for now
	// TODO: support more media types
	if !strings.HasPrefix(page.mediaType, "text/") {
		fmt.Println(ErrorColor("Unsupported type " + page.mediaType))
		return
	}
	if page.mediaType == "text/gemini" {
		rendered := c.ParseGeminiPage(page)
		Pager(rendered, c.conf)
		return
	}
	// other text/* stuff
	Pager(string(page.bodyBytes), c.conf)
}

func (c *Client) ParseGeminiPage(page *Page) string {
	width, _, err := term.GetSize(0)
	if err != nil {
		return ErrorColor("error getting terminal size")
	}
	preformatted := false
	rendered := ""
	body := string(page.bodyBytes)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(line, "\r") {
			line = strings.Trim(line, "\r")
		}
		if strings.HasPrefix(line, "```") {
			preformatted = !preformatted

		} else if preformatted {
			rendered += line + "\n"

		} else if strings.HasPrefix(line, "> ") { // not sure if whitespace after > is mandatory for this
			// appending extra \n here because we want quote blocks to stand out
			// with leading and trailing new lines to distinguish from paragraphs
			// as well as making it clear that it's actually a quote block.
			// NOT doing this anymore!
			// (because it looked bad if quotes are continuous)
			// TODO: remove extra new lines in the end
			rendered += ansiwrap.WrapIndent(line, width, 1, 3) + "\n"

		} else if strings.HasPrefix(line, "* ") { // whitespace after * is mandatory
			// Using width - 3 because of 3 spaces "   " indent at the start
			rendered += "   " + ansiwrap.WrapIndent(strings.Replace(line, "*", "•", 1), width-3, 0, 5) + "\n"

		} else if strings.HasPrefix(line, "#") { // whitespace after #'s are optional for headings as per spec
			rendered += ansiwrap.Wrap(h1Style(line), width) + "\n"

		} else if strings.HasPrefix(line, "##") {
			rendered += ansiwrap.Wrap(h2Style(line), width) + "\n"

		} else if strings.HasPrefix(line, "=>") || (page.u.Scheme == "spartan" && strings.HasPrefix(line, "=:")) {
			originalLine := line
			line = line[2:]
			bits := strings.Fields(line)
			parsedLink, err := url.Parse(bits[0])
			if err != nil {
				continue
			}
			link := page.u.ResolveReference(parsedLink) // link url
			var label string                            // link text
			if len(bits) == 1 {
				label = bits[0]
			} else {
				label = strings.Join(bits[1:], " ")
			}
			if strings.HasPrefix(originalLine, "=:") && page.u.Scheme == "spartan" {
				label += " [INPUT]"
				c.inputLinks = append(c.inputLinks, len(c.links)) // using len(c.links) because it is only appended below so the value from that is just right
			}
			c.links = append(c.links, link.String())
			linkLine := fmt.Sprintf("[%d] ", len(c.links)) + linkStyle(label)
			if link.Scheme != "gemini" {
				linkLine += fmt.Sprintf(" (%s)", link.Scheme)
			}
			rendered += ansiwrap.Wrap(linkLine, width) + "\n"
		} else {
			rendered += ansiwrap.Wrap(line, width) + "\n"
		}
	}
	// Remove last \n
	if len(rendered) > 0 {
		rendered = rendered[:len(rendered)-1]
	}
	return rendered
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
	res, err := SpartanParsedURL(parsed)
	if err != nil {
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	defer (*res.conn).Close()
	c.links = make([]string, 0, 100) // reset links
	c.inputLinks = make([]int, 0, 100)

	page := &Page{bodyBytes: nil, mediaType: "", u: parsed, params: nil}
	// Handle status
	switch res.status {
	case 2:
		mediaType, params, err := ParseMeta(res.meta)
		if err != nil {
			fmt.Println(ErrorColor("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			fmt.Println(ErrorColor("Unable to read body. %s", err))
		}
		page.bodyBytes = bodyBytes
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
	res, err := GeminiParsedURL(*parsed)
	if err != nil {
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	defer res.conn.Close()
	c.links = make([]string, 0, 100) // reset links
	c.inputLinks = make([]int, 0, 100)

	// mediaType and params will be parsed later
	page := &Page{bodyBytes: nil, mediaType: "", u: parsed, params: nil}
	statusGroup := res.status / 10 // floor division
	switch statusGroup {
	case 1:
		fmt.Println(res.meta)
		if res.status == 11 {
			return c.Input(page.u.String(), true) // sensitive input
		}
		return c.Input(page.u.String(), false)
	case 2:
		mediaType, params, err := ParseMeta(res.meta)
		if err != nil {
			fmt.Println(ErrorColor("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			fmt.Println(ErrorColor("Unable to read body. %s", err))
		}
		page.bodyBytes = bodyBytes
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		return c.HandleURL(res.meta) // TODO: max redirect times
	case 4, 5:
		switch res.status {
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
		fmt.Println(ErrorColor("%d %s", res.status, res.meta))
	case 6:
		fmt.Println(res.meta)
		fmt.Println("Sorry, gelim does not support client certificates yet.")
	default:
		fmt.Println(ErrorColor("invalid status code %d", res.status))
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
