package main

import (
	"bufio"
	"crypto/tls"
	"io/ioutil"
	"mime"
	"strconv"
	//"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/lmorg/readline"
	"github.com/manifoldco/ansiwrap"
	"golang.org/x/term"
)

type Response struct {
	status     int
	meta       string // not parsed into mediaType and params yet
	bodyReader *bufio.Reader
}

// these should be used but atm it isnt, lol
//var (
//ErrConnFail       = errors.New("connection failed")
//ErrInvalidStatus  = errors.New("invalid status code")
//ErrDecodeMetaFail = errors.New("failed to decode meta header")
//)

var inputReader = readline.NewInstance()

var (
	h1Style = color.New(color.Bold).Add(color.Underline).Add(color.FgYellow).SprintFunc()
	h2Style = color.New(color.Bold).SprintFunc()
)

// GeminiParsedURL fetches u and displays the page
func GeminiParsedURL(u url.URL) bool {
	host := u.Host
	// Connect to server
	if u.Port() == "" {
		host += ":1965"
	}
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println(ErrorColor("unable to connect to " + u.Host))
		fmt.Println(ErrorColor(err.Error()))
		return false
	}
	defer conn.Close()
	// Send request
	conn.Write([]byte(u.String() + "\r\n"))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	responseHeader, err := reader.ReadString('\n')
	// Parse header
	parts := strings.Fields(responseHeader)
	status, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println(ErrorColor("invalid status code:" + parts[0]))
		return false
	}
	statusGroup := status / 10 // floor division
	meta := strings.Join(parts[1:], " ")
	res := Response{status, meta, reader}

	links = make([]string, 0, 100) // reset links

	switch statusGroup {
	case 1:
		fmt.Println(res.meta)
		if res.status == 11 {
			return Input(u.String(), true) // sensitive input
		}
		return Input(u.String(), false)
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
		GeminiDisplay(bodyBytes, mediaType, u) // does it need params?
	case 3:
		return GeminiURL(res.meta) // TODO: max redirect times
	case 4, 5:
		fmt.Println(ErrorColor("%d %s", res.status, res.meta))
	case 6:
		fmt.Println(res.meta)
	default:
		fmt.Println(ErrorColor("invalid status code %d", res.status))
		return false
	}
	if (len(history) > 0) && (history[len(history)-1] != u) || len(history) == 0 {
		history = append(history, u)
	}
	return true
}

// GeminiURL parses u and calls GeminiParsedURL with the parsed url
func GeminiURL(u string) bool {
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		fmt.Println(ErrorColor("invalid url"))
		return false
	}
	if parsed.Scheme == "" {
		// have to parse again
		// ignoring err since it shouldn't fail here if it succeeded above
		parsed, _ = url.Parse("gemini://" + u)
	}
	if parsed.Scheme != "gemini" {
		fmt.Println(ErrorColor("Unsupported scheme %s", parsed.Scheme))
		return false
	}
	return GeminiParsedURL(*parsed)
}

// GeminiDisplay displays bodyBytes with a pager
func GeminiDisplay(bodyBytes []byte, mediaType string, parsedURL url.URL) {
	// text/* content only for now
	// TODO: support more media types
	if !strings.HasPrefix(mediaType, "text/") {
		fmt.Println(ErrorColor("Unsupported type " + mediaType))
		return
	}
	body := string(bodyBytes)
	if mediaType == "text/gemini" {
		page := GeminiPage(body, parsedURL)
		Pager(page)
		return
	}
	// other text/* stuff
	Pager(body)
}

// GeminiPage returns a rendered gemini page string
func GeminiPage(body string, currentURL url.URL) string {
	width, _, err := term.GetSize(0)
	if err != nil {
		return ErrorColor("error getting terminal size")
	}
	preformatted := false
	page := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "```") {
			preformatted = !preformatted
		} else if preformatted {
			page = page + line + "\n"
		} else if strings.HasPrefix(line, "# ") {
			page += ansiwrap.Wrap(h1Style(line), width) + "\n"
		} else if strings.HasPrefix(line, "## ") {
			page += ansiwrap.Wrap(h2Style(line), width) + "\n"
		} else if strings.HasPrefix(line, "=>") {
			line = line[2:]
			bits := strings.Fields(line)
			parsedLink, err := url.Parse(bits[0])
			if err != nil {
				continue
			}
			link := currentURL.ResolveReference(parsedLink) // link url
			var label string                                // link text
			if len(bits) == 1 {
				label = bits[0]
			} else {
				label = strings.Join(bits[1:], " ")
			}
			links = append(links, link.String())
			if link.Scheme != "gemini" {
				page += ansiwrap.Wrap(fmt.Sprintf("[%d %s] %s\n", len(links), link.Scheme, label), width) + "\n"
				continue
			}
			page += ansiwrap.Wrap(fmt.Sprintf("[%d] %s\n", len(links), label), width) + "\n"
		} else {
			page += ansiwrap.Wrap(line, width) + "\n"
		}
	}
	page = page[:len(page)-1] // remove last \n
	return page
}

// ParseMeta returns the output of mime.ParseMediaType, but handles the empty
// META which is equal to "text/gemini; charset=utf-8" according to the spec.
func ParseMeta(meta string) (string, map[string]string, error) {
	if meta == "" {
		return "text/gemini", make(map[string]string), nil
	}

	mediatype, params, err := mime.ParseMediaType(meta)
	if mediatype != "" && err != nil {
		// The mediatype was successfully decoded but there's some error with the params
		// Ignore the params
		return mediatype, make(map[string]string), nil
	}
	return mediatype, params, err
}

// Input handles Input status codes
func Input(u string, sensitive bool) (ok bool) {
	inputReader.SetPrompt("INPUT> ")
	if sensitive {
		inputReader.PasswordMask = '*'
		oldHistory := inputReader.History
		inputReader.History = new(readline.NullHistory)
		defer func() { inputReader.PasswordMask = 0; inputReader.History = oldHistory }()
	}
	query, err := inputReader.Readline()
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
	return GeminiURL(u)
}
