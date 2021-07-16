package main

import (
	"fmt"
	"github.com/lmorg/readline"
	"github.com/manifoldco/ansiwrap"
	"golang.org/x/term"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
)

type Page struct {
	bodyBytes []byte
	mediaType string
	params    map[string]string
	u         *url.URL
}

type Client struct {
	links       []string
	inputLinks  []int // contains index to links in `links` that needs spartan input
	history     []*url.URL
	conf        *Config
	inputReader *readline.Instance
	mainReader  *readline.Instance
}

func NewClient() *Client {
	var c Client
	// load config
	conf, err := LoadConfig()
	if err != nil {
		fmt.Println(ErrorColor("Error loading config: %s", err.Error()))
		os.Exit(1)
	}
	// c.history = make([]*url.URL, 100)
	c.links = make([]string, 100)
	c.conf = conf
	c.mainReader = readline.NewInstance()
	c.mainReader.SetPrompt(promptColor(c.conf.Prompt) + "> ")
	c.inputReader = readline.NewInstance()
	return &c
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
			// TODO: remove extra new lines in the end
			rendered += "\n" + quoteStyle(ansiwrap.WrapIndent(line, width, 0, 2)) + "\n\n"

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
			if strings.HasPrefix(originalLine, "=:") {
				label += " [INPUT]"
				c.inputLinks = append(c.inputLinks, len(c.links)) // using len(.clinks) because it is only appended below so the value from that is just right
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
	rendered = rendered[:len(rendered)-1] // remove last \n
	return rendered
}

// Input handles Input status codes
func (c *Client) Input(u string, sensitive bool) (ok bool) {
	c.inputReader.SetPrompt("INPUT> ")
	if sensitive {
		c.inputReader.PasswordMask = '*'
		oldHistory := c.inputReader.History
		c.inputReader.History = new(readline.NullHistory)
		defer func() { c.inputReader.PasswordMask = 0; c.inputReader.History = oldHistory }()
	}
	query, err := c.inputReader.Readline()
	if err != nil {
		if err == readline.CtrlC {
			fmt.Println(ErrorColor("\ninput cancelled"))
			return false
		}
		fmt.Println(ErrorColor("\nerror reading input:"))
		fmt.Println(ErrorColor(err.Error()))
		return false
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
			fmt.Println(ErrorColor("Unable to parse header meta\"", res.meta, "\":", err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			fmt.Println(ErrorColor("Unable to read body.", err))
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
			fmt.Println(ErrorColor("Unable to parse header meta\"", res.meta, "\":", err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			fmt.Println(ErrorColor("Unable to read body.", err))
		}
		page.bodyBytes = bodyBytes
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		return c.HandleURL(res.meta) // TODO: max redirect times
	case 4, 5:
		fmt.Println(ErrorColor("%d %s", res.status, res.meta))
	case 6:
		fmt.Println(res.meta)
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