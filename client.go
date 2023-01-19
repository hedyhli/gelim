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

// Page is the structure of a fetched resource
type Page struct {
	bodyBytes []byte
	mediaType string
	params    map[string]string
	u         *url.URL
}

// Client contains all the data for a gelim session
type Client struct {
	links            []string
	inputLinks       []int // contains index to links in `links` that needs spartan input
	history          []*url.URL
	conf             *Config
	style            *Style
	mainReader       *ln.State
	inputReader      *ln.State
	promptHistory    *os.File
	inputHistory     *os.File
	promptSuggestion string

	tourLinks []string // List of links to tour
	tourNext  int      // The index for link that will be visit next time user uses tour
}

// NewClient loads the config file and returns a new client object
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
	c.style = &DefaultStyle // TODO: config styles
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

// QuitClient cleans up opened files and resources, saves history, and calls
// os.Exit with the given status code
func (c *Client) QuitClient(code int) {
	c.mainReader.WriteHistory(c.promptHistory)
	c.inputReader.WriteHistory(c.inputHistory)
	c.promptHistory.Close()
	c.inputHistory.Close()
	c.inputReader.Close()
	c.mainReader.Close()
	os.Exit(code)
}

// GetLinkFromIndex retrieves the link on the current page
func (c *Client) GetLinkFromIndex(i int) (link string, spartanInput bool) {
	spartanInput = false
	if len(c.links) < i {
		c.style.ErrorMsg(fmt.Sprintf("invalid link index, I have %d links so far", len(c.links)))
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

// DisplayPage renders a given page object in the client
func (c *Client) DisplayPage(page *Page) {
	// TODO: proper stream - read the reader and stuff
	if page.mediaType == "application/octet-stream" {
		Pager(string(page.bodyBytes), c.conf)
		return
	}
	// text/* content only for now
	// TODO: support more media types
	if !strings.HasPrefix(page.mediaType, "text/") {
		c.style.ErrorMsg("Unsupported type " + page.mediaType)
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

// ParseGeminiPage parses bytes in page in returns a rendered string for the
// page
func (c *Client) ParseGeminiPage(page *Page) string {
	var (
		h1Style    = c.style.gmiH1.Sprint
		h2Style    = c.style.gmiH2.Sprint
		h3Style    = c.style.gmiH3.Sprint
		preStyle   = c.style.gmiPre.Sprint
		linkStyle  = c.style.gmiLink.Sprint
		quoteStyle = c.style.gmiQuote.Sprint
	)

	termWidth, _, err := term.GetSize(0)
	if err != nil {
		// TODO do something
		c.style.ErrorMsg("Error getting terminal size")
		return ""
	}
	sides := int(float32(termWidth) * c.conf.LeftMargin)
	width := termWidth - sides
	if width > c.conf.MaxWidth {
		width = c.conf.MaxWidth
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
			rendered += strings.Repeat(" ", sides) + preStyle(line) + "\n"

		} else if strings.HasPrefix(line, "> ") { // not sure if whitespace after > is mandatory for this
			// appending extra \n here because we want quote blocks to stand out
			// with leading and trailing new lines to distinguish from paragraphs
			// as well as making it clear that it's actually a quote block.
			// NOT doing this anymore!
			// (because it looked bad if quotes are continuous)
			// TODO: remove extra new lines in the end
			rendered += ansiwrap.GreedyIndent(quoteStyle(line), width, 1+sides, 3+sides) + "\n"

		} else if strings.HasPrefix(line, "* ") { // whitespace after * is mandatory
			// Using width - 3 because of 3 spaces "   " indent at the start
			rendered += "   " + ansiwrap.GreedyIndent(strings.Replace(line, "*", "•", 1), width-3, sides, 5+sides) + "\n"

		} else if strings.HasPrefix(line, "###") {
			rendered += ansiwrap.GreedyIndent(h3Style(line), width, sides, sides) + "\n"
		} else if strings.HasPrefix(line, "##") {
			rendered += ansiwrap.GreedyIndent(h2Style(line), width, sides, sides) + "\n"
		} else if strings.HasPrefix(line, "#") { // whitespace after #'s are optional for headings as per spec
			rendered += ansiwrap.GreedyIndent(h1Style(line), width, sides, sides) + "\n"

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

			c.links = append(c.links, link.String())
			linkLine := fmt.Sprintf("[%d] ", len(c.links))
			leftWidth := len(linkLine) // Used when wrapping below
			linkLine += linkStyle(label)

			// Format the link so that when it wraps the rest indent is after the [%d]:
			//    [10] foo bar baz. I am the first line of the link
			//         I am wrapped from the link
			//
			// Or for ones that are a single word:
			//    [10] gemini://super-duper-long-host.site/super-lo
			//         ng-url/slug/path/to/file.gmi

			// So if the label is a single word
			if !strings.Contains(label, " ") {
				// We special-case links where the label is the literal link
				// (no label) or the link text is a single long word, because
				// ansiwrap doesn't handle that.
				if len(linkLine) > width {
					// Quite a clumsy but simple wrapping algorithm that
					// doesn't care about the word splits because, hey, our
					// whole link is a word ;P

					// Wraps a given wordby a given length and takes care of
					// indentation for gelim page displays.
					restIndent := strings.Repeat(" ", sides+leftWidth+1)

					newLinkLine := strings.Repeat(" ", sides) // First indent
					newLinkLine += linkLine[:width] + "\n"    // Add in initial chunk first

					llen := len(linkLine)
					start := width - 1

					// Loop through each `width` and build up newLinkLine on
					// each iteration.

					// It had been a while since I first wrote this and when I
					// committed this. In other words I forgot how this worked,
					// but it seems to work ok so I won't be touching it until
					// I have time to remember how this worked.
					for end := width + width; ; end += width {
						if end >= llen {
							// End
							newLinkLine += restIndent + linkLine[start:]
							break
						}
						newLinkLine += restIndent + linkLine[start:end] + "\n"
						start += width
					}

					linkLine = newLinkLine
				} else {
					// If this single worded link length is less than desired width
					// Don't wrap if it doesn't need wrapping
					linkLine = strings.Repeat(" ", sides) + linkLine
				}
			}
			// Spartan input label
			if strings.HasPrefix(originalLine, "=:") && page.u.Scheme == "spartan" {
				linkLine += " [INPUT]"
				// c.inputLinks is 0-indexed
				c.inputLinks = append(c.inputLinks, len(c.links)-1)
			}

			// TODO: Config for protocols that appends `(protocol-name)` at the
			// end of link
			if link.Scheme != "gemini" {
				linkLine += fmt.Sprintf(" (%s)", link.Scheme)
			}
			// XXX: wrap twice for single word

			linkLine = ansiwrap.GreedyIndent(linkLine, width, sides, sides+leftWidth)
			if len(c.links) < 10 {
				linkLine = " " + linkLine
			}
			rendered += linkLine + "\n"
		} else {
			// Normal paragraph
			rendered += ansiwrap.GreedyIndent(line, width, sides, sides) + "\n"
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
			fmt.Println()
			c.style.WarningMsg("Input cancelled")
			return false
		}
		fmt.Println()
		c.style.ErrorMsg("Error reading input: " + err.Error())
		return false
	}
	if !sensitive {
		c.inputReader.AppendHistory(query)
	}
	u = u + "?" + queryEscape(query)
	return c.HandleURL(u)
}

// HandleURL parses the URL, then calls HandleParsedURL. It returns whether it
// was a valid URL
func (c *Client) HandleURL(u string) bool {
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		c.style.ErrorMsg("Invalid url")
		return false
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		// have to parse again
		parsed, err = url.Parse("gemini://" + u)
		if err != nil {
			c.style.ErrorMsg("Invalid url")
			return false
		}
	}
	return c.HandleParsedURL(parsed)
}

// Handles either a spartan URL or a gemini URL
func (c *Client) HandleParsedURL(parsed *url.URL) bool {
	// TODO; config proxies or program to do other shemes
	if parsed.Scheme != "gemini" && parsed.Scheme != "spartan" {
		c.style.ErrorMsg("Unsupported scheme " + parsed.Scheme)
		return false
	}
	if parsed.Scheme == "gemini" {
		return c.HandleGeminiParsedURL(parsed)
	}
	return c.HandleSpartanParsedURL(parsed)
}

// HandleSpartanParsedURL makes an requested to parsed URL, displays the page,
// and returns whether it was successful.
func (c *Client) HandleSpartanParsedURL(parsed *url.URL) bool {
	res, err := SpartanParsedURL(parsed)
	if err != nil {
		c.style.ErrorMsg(err.Error())
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
			c.style.ErrorMsg(fmt.Sprintf("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			c.style.ErrorMsg("Unable to read body: " + err.Error())
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

// HandleGeminiParsedURL makes an requested to parsed URL, displays the page,
// and returns whether it was successful.
func (c *Client) HandleGeminiParsedURL(parsed *url.URL) bool {
	res, err := GeminiParsedURL(*parsed)
	if err != nil {
		c.style.ErrorMsg(err.Error())
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
		u := strings.TrimRight(page.u.String(), "?"+page.u.RawQuery)
		fmt.Println(res.meta)
		if res.status == 11 {
			return c.Input(u, true) // sensitive input
		}
		return c.Input(u, false)
	case 2:
		mediaType, params, err := ParseMeta(res.meta)
		if err != nil {
			c.style.ErrorMsg(fmt.Sprintf("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			c.style.ErrorMsg("Unable to read body: " + err.Error())
		}
		page.bodyBytes = bodyBytes
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		return c.HandleURL(res.meta) // TODO: max redirect times
	case 4, 5:
		// TODO: use res.meta
		switch res.status {
		case 40:
			c.style.ErrorMsg("Temperorary failure")
		case 41:
			c.style.ErrorMsg("Server unavailable")
		case 42:
			c.style.ErrorMsg("CGI error")
		case 43:
			c.style.ErrorMsg("Proxy error")
		case 44:
			c.style.ErrorMsg("Slow down")
		case 52:
			c.style.ErrorMsg("Gone")
		}
		c.style.ErrorMsg(fmt.Sprintf("%d %s", res.status, res.meta))
	case 6:
		fmt.Println(res.meta)
		fmt.Println("Sorry, gelim does not support client certificates yet.")
	default:
		c.style.ErrorMsg(fmt.Sprintf("Invalid status code %d", res.status))
		return false
	}
	if (len(c.history) > 0) && (c.history[len(c.history)-1].String() != parsed.String()) || len(c.history) == 0 {
		c.history = append(c.history, parsed)
	}
	return true
}

// Search opens the SearchURL in config with query-escaped query
func (c *Client) Search(query string) {
	u := c.conf.SearchURL + "?" + queryEscape(query)
	c.HandleURL(u)
}

// LookupCommand attempts to get the corresponding command from cmdStr,
// returning the command and whether the command was found
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

// Command attempts to execute a given gelim command and returns whether the
// command was found
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
