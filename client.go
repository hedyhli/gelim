package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"git.sr.ht/~adnano/go-xdg"
	"github.com/google/shlex"
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

type RedirectInfo struct {
	history []string
	// Total length of the history slice (10 if c.MaxRedirects <- 0). We cap it
	// at 10 to prevent it from infinetely overflowing, effectively we store
	// only the last 10 redirect URLs, hence user only see those last 10.
	historyCap int
	// Number of elems in redir history
	// that is occupied. Also used as
	// index.
	historyLen int
	// Total number of redirects made. >= historyLen
	count int

	showHistory func()
	reset       func()
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

	lastPage string

	redir *RedirectInfo // The object itself does not get changed, only attributes in it -- throughout the runtime of gelim
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

	c.redir = &RedirectInfo{historyCap: conf.MaxRedirects, historyLen: 0}
	if c.redir.historyCap <= 0 {
		c.redir.historyCap = 10
	}
	c.redir.history = make([]string, c.redir.historyCap)
	c.redir.showHistory = func() {
		for i := 0; i < c.redir.historyLen; i++ {
			fmt.Println(i+1, c.redir.history[i])
		}
	}
	c.redir.reset = func() {
		// Reset redirects
		c.redir.count = 0
		c.redir.historyLen = 0

		// Not initializing new slice with make() so we don't rely too much on GC.
		// Initial c.redir.historyCap is ideally maintained.
		for i := range c.redir.history {
			c.redir.history[i] = ""
		}
	}
	// note that the c.redir.history slice is initialized at HandleURLWrapper

	c.conf = conf
	c.style = &DefaultStyle // TODO: config styles
	c.lastPage = ""
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
	if len(c.links) < i || i < 1 {
		c.style.ErrorMsg(fmt.Sprintf("Link index argument out of range. There are %d links on the page", len(c.links)))
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
		c.lastPage = string(page.bodyBytes)
		Pager(c.lastPage, c.conf)
		return
	}
	if page.mediaType == "nex/directory" {
		// The directory listings in Nex is like gemtext except it's all plain
		// text, only "=>" links are parsed.
		rendered := c.ParseNexDirectoryPage(page)
		c.lastPage = rendered
		Pager(c.lastPage, c.conf)
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
		c.lastPage = rendered
		Pager(c.lastPage, c.conf)
		return
	}
	// other text/* stuff
	c.lastPage = string(page.bodyBytes)
	Pager(c.lastPage, c.conf)
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
	width := termWidth
	sides := 0
	if width > c.conf.MaxWidth {
		width = c.conf.MaxWidth
		// sides := int((termWidth - width) / 2)
		// XXX: Huh?
		sides = termWidth - width
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
			line = strings.TrimSpace(line[2:])
			if line == "" {
				// Empty link line
				rendered += strings.Repeat(" ", sides) + originalLine + "\n"
				continue
			}
			bits := strings.Fields(line)
			parsedLink, err := url.Parse(bits[0])
			if err != nil {
				// FIXME: not adding to rendered?
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
	return c.HandleURLWrapper(u)
}

// PromptRedirect asks for input on whether to follow a redirect. Return user's
// choice and whether the prompt was successful (in that order!).
func (c *Client) PromptRedirect(nextDest string) (opt bool, ok bool) {
	ok = true

	if c.conf.ShowRedirectHistory {
		c.redir.showHistory()
		fmt.Println()
	}

	fmt.Println("Redirect to:")
	fmt.Println(nextDest)

	for { // Our good old 'prompt until valid' structure ;P
		optStr, err := c.inputReader.PromptWithSuggestion("[y/n]> ", "", 1)

		if err != nil {
			opt = false
			if err == ln.ErrPromptAborted || err == io.EOF {
				fmt.Println()
				// ok is true here
				c.style.WarningMsg("Cancelled")
				return
			}
			ok = false
			fmt.Println()
			c.style.ErrorMsg("Error reading input: " + err.Error())
			return
		}

		optStr = strings.ToLower(optStr)

		switch optStr {
		case "y":
			opt = true
		case "n":
			opt = false
		default:
			c.style.ErrorMsg("Please input y or n only.")
			continue
		}
		break
	}
	return
}

// RedirectURL handles a redirect by checking MaxRedirects and calling PromptRedirect
func (c *Client) RedirectURL(u string) (ok bool) {
	var opt = true
	var promptCalled = false
	ok = true

	if c.conf.MaxRedirects == 0 {
		// Option to prompt for all redirects
		opt, ok = c.PromptRedirect(u)
	} else if c.conf.MaxRedirects > 0 && c.conf.MaxRedirects <= c.redir.count {
		c.style.WarningMsg(fmt.Sprintf("Max redirects of %d reached", c.redir.count))
		opt, ok = c.PromptRedirect(u)
		promptCalled = true
	} // for MaxRedidrects set to negative value, follow all redirects

	if !ok || !opt {
		return false
	}

	if promptCalled {
		// Say max redirects is set to 2. User visits a link. Gets redirected 2
		// times. gelim prompts whether to follow the next redirect. User
		// inputs yes. Then gelim must reset the redirects as if user is
		// visiting a fresh new links, so that the next 2 redirects (if any)
		// should be handled automatically as before.
		//
		// So if the URL was to redirect the user a total of 4 times and max
		// redirects conf is set to 2, the user will be prompted only 2 times.
		// Once after first two redirects, another time after the next 2
		// redirects.
		c.redir.reset()
		return c.HandleURL(u)
	}

	c.redir.count += 1
	if c.redir.historyLen+1 > len(c.redir.history) && c.conf.MaxRedirects <= 0 {
		// This should not happen if c.conf.MaxRedirects > 0.
		//
		// If 10 redirects are reached we use the rolling window, effectively
		// c.redir.history will always only contain the 10 MOST RECENT
		// redirects. Older ones are discarded
		// XXX: Is this memory safe/efficient?
		c.redir.history = c.redir.history[1:]
		c.redir.history = append(c.redir.history, u)

		if c.redir.count >= 20 {
			// XXX: Can redirects be implmented without recursion?
			c.style.ErrorMsg("The URL redirected you 20 times. Stack overflow may be reached soon, aborting.")
			fmt.Println("Here are the", c.redir.historyLen, "most recent redirects.")
			c.redir.showHistory()
			return false
		}
	} else {
		c.redir.historyLen += 1
		c.redir.history[c.redir.historyLen-1] = u // -1 due to 0-indexing
	}
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

// HandleURLWrapper is like HandleURL but should only be used for the first
// request
//
// It sets c.redir.count and c.redir.historyLen to 0 before calling c.HandleURL
// with the same argument(s).
func (c *Client) HandleURLWrapper(u string) bool {
	c.redir.reset()
	return c.HandleURL(u)
}

// Handles either a spartan URL, Nex, or a gemini URL
func (c *Client) HandleParsedURL(parsed *url.URL) bool {
	// TODO; config proxies or program to do other shemes
	if parsed.Scheme != "gemini" && parsed.Scheme != "spartan" && parsed.Scheme != "nex" {
		c.style.ErrorMsg("Unsupported protocol " + parsed.Scheme)
		fmt.Println("URL:", parsed)
		return false
	}
	if parsed.Scheme == "gemini" {
		return c.HandleGeminiParsedURL(parsed)
	}
	if parsed.Scheme == "nex" {
		return c.HandleNexParsedURL(parsed)
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
		// Only reset links if the page is a success
		c.links = make([]string, 0, 100) // reset links
		c.inputLinks = make([]int, 0, 100)

		page.bodyBytes = bodyBytes
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		return c.RedirectURL("spartan://" + parsed.Host + res.meta)
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

// HandleNexParsedURL makes an requested to parsed URL, displays the page,
// and returns whether it was successful.
func (c *Client) HandleNexParsedURL(parsed *url.URL) bool {
	res, err := NexParsedURL(parsed)
	if err != nil {
		c.style.ErrorMsg(err.Error())
		return false
	}
	defer (*res.conn).Close()

	page := &Page{bodyBytes: nil, mediaType: "", u: parsed, params: nil}
	bodyBytes, err := ioutil.ReadAll(res.bodyReader)
	if err != nil {
		c.style.ErrorMsg("Unable to read body: " + err.Error())
	}
	// Only reset links if the page is a success
	c.links = make([]string, 0, 100) // reset links
	c.inputLinks = make([]int, 0, 100)

	page.bodyBytes = bodyBytes

	// TODO: check file extension
	if res.fileExt == "/" {
		page.mediaType = "nex/directory"
	} else {
		// Assume plain text for now
		page.mediaType = "text/plain"
	}
	c.DisplayPage(page)

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

	// mediaType and params will be parsed later
	page := &Page{bodyBytes: nil, mediaType: "", u: parsed, params: nil}
	statusGroup := res.status / 10 // floor division
	statusRightDigit := res.status - statusGroup*10
	switch statusGroup {
	case 1:
		if statusRightDigit > 1 {
			c.style.WarningMsg(fmt.Sprintf("Undefined status code %v", res.status))
		}

		u := strings.TrimRight(page.u.String(), "?"+page.u.RawQuery)
		fmt.Println(res.meta)
		if res.status == 11 {
			return c.Input(u, true) // sensitive input
		}
		return c.Input(u, false)
	case 2:
		if statusRightDigit > 0 {
			c.style.WarningMsg(fmt.Sprintf("Undefined status code %v", res.status))
		}

		mediaType, params, err := ParseMeta(res.meta)
		if err != nil {
			c.style.ErrorMsg(fmt.Sprintf("Unable to parse header meta\"%s\": %s", res.meta, err))
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			c.style.ErrorMsg("Unable to read body: " + err.Error())
		}

		// Only reset links if the page is a success
		c.links = make([]string, 0, 100) // reset links
		c.inputLinks = make([]int, 0, 100)

		page.bodyBytes = bodyBytes
		page.mediaType = mediaType
		page.params = params
		c.DisplayPage(page)
	case 3:
		if statusRightDigit > 1 {
			c.style.WarningMsg(fmt.Sprintf("Undefined status code %v", res.status))
		}
		// TODO: permanent vs temporary redir

		if res.meta == "" {
			c.style.ErrorMsg(fmt.Sprintf("Redirect status code %d with no redirect URL returned by server.", res.status))
			return false
		}
		return c.RedirectURL(res.meta)
	case 4, 5:
		// TODO: use res.meta
		c.style.WarningMsg("The server responded with an erroneous status:")
		// switch res.status {
		// case 40:
		// 	c.style.ErrorMsg("Temperorary failure")
		// case 41:
		// 	c.style.ErrorMsg("Server unavailable")
		// case 42:
		// 	c.style.ErrorMsg("CGI error")
		// case 43:
		// 	c.style.ErrorMsg("Proxy error")
		// case 44:
		// 	c.style.ErrorMsg("Slow down")
		// case 52:
		// 	c.style.ErrorMsg("Gone")
		// }
		c.style.WarningMsg(fmt.Sprintf("%d %s", res.status, res.meta))
		if statusGroup == 4 && statusRightDigit > 4 || statusGroup == 5 && (statusRightDigit > 3 && statusRightDigit != 9) {
			c.style.WarningMsg(fmt.Sprintf("Undefined status code %v", res.status))
		}

	case 6:
		if statusRightDigit > 2 {
			c.style.WarningMsg(fmt.Sprintf("Undefined status code %v", res.status))
		}
		fmt.Println(res.meta)
		fmt.Println("Sorry, gelim does not support client certificates yet.")
	default:
		c.style.ErrorMsg(fmt.Sprintf("Invalid status code %d", res.status))
		// return false
	}
	if (len(c.history) > 0) && (c.history[len(c.history)-1].String() != parsed.String()) || len(c.history) == 0 {
		c.history = append(c.history, parsed)
	}
	return true
}

// Search opens the SearchURL in config with query-escaped query
func (c *Client) Search(query string) {
	u := c.conf.SearchURL + "?" + queryEscape(query)
	c.HandleURLWrapper(u)
}

////// Command stuff //////

// LookupCommand attempts to get the corresponding command from cmdStr,
// returning the command and whether the command was found. Does not repect
// meta commands
func (c *Client) LookupCommand(cmdStr string) (cmdName string, cmd Command, ok bool) {
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

// LookupCommandWithMeta does the same as LookupCommand but it respects metaCommands.
//
// LookupCommandWithMeta attempts to resolve cmdStr into the proper command,
// respecting meta commands.
func (c *Client) LookupCommandWithMeta(cmdStr string) (cmd Command, ok bool) {
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
		cmd = metaCommands[cmdName]
		ok = true
		return
	}
	// Not a meta command, then:
	_, cmd, ok = c.LookupCommand(cmdStr)
	if !ok {
		return
	}
	// below logic is moved to places where LookupCommandWithMeta is called to
	// (counter-intuitively) remove duplication.
	// "<cmd> help"
	// if (firstArg == "help" || firstArg == "?" || firstArg == "--help") {
	// 	return c.LookupCommandWithMeta("help", cmdStr)
	// }
	return
}

// Command uses LookupCommandWithMeta to search for the appropriate command
// then runs it
func (c *Client) Command(cmdStr string, args ...string) (ok bool) {
	var cmd Command

	if len(args) > 0 && (args[0] == "help" || args[0] == "?" || args[0] == "--help") {
		ok = true
		metaCommands["help"].do(c, cmdStr)
		return
	}

	cmd, ok = c.LookupCommandWithMeta(cmdStr)
	if !ok {
		return
	}
	cmd.do(c, args...)
	return
}

// GetCommandAndArgs parses a command line string, looks up using
// LookupCommandWithMeta, then splits arguments respecting the comamnd's
// quotedArgs field.
//
// Returns ok = false if the command is not found
func (c *Client) GetCommandAndArgs(line string) (
	cmd Command, cmdStr string, args []string, ok bool,
) {

	// Split by spaces by default
	lineFields := strings.Split(line, " ")

	// Command and the rest of the line is always separated by a space
	cmdStr = lineFields[0]
	if len(lineFields) > 1 {
		args = lineFields[1:]
	}

	if len(args) > 0 &&
		(args[0] == "help" || args[0] == "?" || args[0] == "--help") {

		ok = true
		cmd = metaCommands["help"]
		// Discarding the rest of the arguments, if any. Because it may be used
		// confused with "help cmd1 cmd2 cmd3"
		args = []string{cmdStr}

		cmdStr = "help"
		return
	}

	cmd, ok = c.LookupCommandWithMeta(cmdStr)
	if !ok || !cmd.quotedArgs {
		return
	}

	// Rejoin args, split using shlex
	// XXX: err is ignored
	args, _ = shlex.Split(strings.Join(args, " "))
	return
}
