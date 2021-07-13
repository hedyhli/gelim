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
	u         *url.URL
}

type Client struct {
	links       []string
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

func (c *Client) GetLinkFromIndex(i int) string {
	if len(c.links) < i {
		fmt.Println(ErrorColor("invalid link index, I have %d links so far", len(c.links)))
		return ""
	}
	return c.links[i-1]
}

func (c *Client) DisplayPage(page *Page) {
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
			rendered += "\n" + quoteStyle(ansiwrap.WrapIndent(line, width, 0, 2)) + "\n\n"

		} else if strings.HasPrefix(line, "* ") { // whitespace after * is mandatory
			rendered += ansiwrap.WrapIndent(strings.Replace(line, "*", "•", 1), width, 0, 2) + "\n"

		} else if strings.HasPrefix(line, "#") { // whitespace after #'s are optional for headings as per spec
			rendered += ansiwrap.Wrap(h1Style(line), width) + "\n"

		} else if strings.HasPrefix(line, "##") {
			rendered += ansiwrap.Wrap(h2Style(line), width) + "\n"

		} else if strings.HasPrefix(line, "=>") {
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
			if link.Scheme != "gemini" {
				rendered += ansiwrap.Wrap(linkStyle("[%d %s] %s", len(c.links), link.Scheme, label), width) + "\n"
				continue
			}
			rendered += ansiwrap.Wrap(linkStyle("[%d] %s", len(c.links), label), width) + "\n"
		} else {
			rendered += ansiwrap.Wrap(line, width) + "\n"
		}
	}
	rendered = rendered[:len(rendered)-1] // remove last \n
	return rendered
}

// Input handles Input status codes
func (c *Client) Input(page *Page, sensitive bool) (ok bool) {
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
	u := page.u.String() + "?" + queryEscape(query)
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
	if parsed.Scheme != "gemini" {
		fmt.Println(ErrorColor("Unsupported scheme %s", parsed.Scheme))
		return false
	}
	return c.HandleParsedURL(parsed)
}

func (c *Client) HandleParsedURL(parsed *url.URL) bool {
	res, err := GeminiParsedURL(*parsed)
	if err != nil {
		fmt.Println(ErrorColor(err.Error()))
	}
	defer res.conn.Close()
	c.links = make([]string, 0, 100) // reset links

	page := &Page{bodyBytes: nil, mediaType: "", u: parsed}
	statusGroup := res.status / 10 // floor division
	switch statusGroup {
	case 1:
		fmt.Println(res.meta)
		if res.status == 11 {
			return c.Input(page, true) // sensitive input
		}
		return c.Input(page, true)
	case 2:
		mediaType, _, err := ParseMeta(res.meta) // what to do with params
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

func (c *Client) Command(cmdStr string, args ...string) bool {
	cmd := ""
	for name, v := range commands {
		if name == cmdStr {
			cmd = name
			break
		}
		for _, alias := range v.aliases {
			if alias == cmdStr {
				cmd = name
				break
			}
		}
	}
	if cmd == "" {
		return false
	}
	commands[cmd].do(c, args...)
	return true
}
