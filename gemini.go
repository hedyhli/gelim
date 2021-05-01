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

// GeminiURL takes url as a string, fetches it, and displays it
func GeminiURL(u string) bool {
	if !strings.HasPrefix(u, "gemini://") {
		u = "gemini://" + u
	}
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		fmt.Println(ErrorColor("invalid url"))
		return false
	}
	// Connect to server
	host := parsed.Host
	if parsed.Port() == "" {
		host += ":1965"
	}
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println(ErrorColor("unable to connect to " + parsed.Host))
		fmt.Println(ErrorColor(err))
		return false
	}
	defer conn.Close()
	// Send request
	conn.Write([]byte(parsed.String() + "\r\n"))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	responseHeader, err := reader.ReadString('\n')
	// Parse header
	parts := strings.Fields(responseHeader)
	status, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println(ErrorColor("invalid status code:", parts[0]))
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
			return Input(u, true) // sensitive input
		}
		return Input(u, false)
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
		GeminiDisplay(bodyBytes, mediaType, *parsed) // does it need params?
	case 3:
		return GeminiURL(res.meta) // TODO: max redirect times
	case 4, 5:
		fmt.Println(res.meta)
	case 6:
		fmt.Println("im not good enough in go to implement certs lol")
	default:
		fmt.Println(ErrorColor("invalid status code:", res.status))
		return false
	}
	if (len(history) > 0) && (history[len(history)-1] != u) || len(history) == 0 {
		history = append(history, u)
	}
	return true
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
	preformatted := false
	page := ""
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			page += h1Style(line)
		} else if strings.HasPrefix(line, "## ") {
			page += h2Style(line)
		} else if strings.HasPrefix(line, "```") {
			preformatted = !preformatted
		} else if preformatted {
			page = page + line + "\n"
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
				label = link.String()
			} else {
				label = strings.Join(bits[1:], " ")
			}
			links = append(links, link.String())
			if link.Scheme != "gemini" {
				page = page + fmt.Sprintf("[%d %s] %s\n", len(links), link.Scheme, label) + "\n"
				continue
			}
			page = page + fmt.Sprintf("[%d] %s\n", len(links), label) + "\n"
		} else {
			// This should really be wrapped, but there's
			// no easy support for this in Go's standard
			// library (says solderpunk)
			//fmt.Println(line)
			page = page + line + "\n"
		}
	}
	page = page[:len(page)-2] // remove last \n
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
		fmt.Println(ErrorColor(err))
		return false
	}
	u = u + "?" + queryEscape(query)
	return GeminiURL(u)
}
