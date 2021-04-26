package main

import (
	"bufio"
	"crypto/tls"
	//"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/MarekStancik/readline"
	flag "github.com/spf13/pflag"
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

var (
	links   []string = make([]string, 0, 100)
	history []string = make([]string, 0, 100)
)

var noInteractive = flag.BoolP("no-interactive", "I", false, "don't go to the line-mode interface\n")
var appendInput = flag.StringP("input", "i", "", "append input to URL ('?' + percent-encoded input)\n")
var helpFlag = flag.BoolP("help", "h", false, "get help on the cli")

func printHelp() {
	fmt.Println("just enter a url to start browsing...")
	fmt.Println()
	fmt.Println("commands")
	fmt.Println("  b        go back")
	fmt.Println("  q, x     quit")
	fmt.Println("  history  view history")
	fmt.Println("  r        reload")
	fmt.Println("\nenter number to go to a link")
}

func displayGeminiPage(body string, currentURL url.URL) {
	preformatted := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "```") {
			preformatted = !preformatted
		} else if preformatted {
			fmt.Println(line)
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
				fmt.Printf("[%d %s] %s\n", len(links), link.Scheme, label)
				continue
			}
			fmt.Printf("[%d] %s\n", len(links), label)
		} else {
			// This should really be wrapped, but there's
			// no easy support for this in Go's standard
			// library (says solderpunk)
			fmt.Println(line)
		}
	}
}

// input handles input status codes
func input(u string) (ok bool) {
	stdinReader := bufio.NewReader(os.Stdin)
	fmt.Print("INPUT> ")
	query, _ := stdinReader.ReadString('\n')
	query = strings.TrimSpace(query)
	u = u + "?" + strings.Replace(url.QueryEscape(query), "+", "%20", -1)
	return urlHandler(u)
}

// parseMeta returns the output of mime.ParseMediaType, but handles the empty
// META which is equal to "text/gemini; charset=utf-8" according to the spec.
func parseMeta(meta string) (string, map[string]string, error) {
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

// displayBody handles the displaying of body bytes for response
func displayBody(bodyBytes []byte, mediaType string, parsedURL url.URL) {
	// text/* content only for now
	// TODO: support more media types
	if !strings.HasPrefix(mediaType, "text/") {
		fmt.Println("Unsupported type " + mediaType)
		return
	}
	body := string(bodyBytes)
	if mediaType == "text/gemini" {
		displayGeminiPage(body, parsedURL)
	} else {
		// Just print any other kind of text
		fmt.Print(body)
	}
}

func urlHandler(u string) bool {
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		fmt.Println("invalid url")
		return false
	}
	// Connect to server
	conn, err := tls.Dial("tcp", parsed.Host+":1965", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println("unable to connect to", parsed.Host, ":", err)
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
	status, err := strconv.Atoi(parts[0][0:1])
	if err != nil {
		fmt.Println("invalid status code:", parts[0][0:1])
		return false
	}
	meta := strings.Join(parts[1:], " ")
	res := Response{status, meta, reader}

	links = make([]string, 0, 100) // reset links

	switch res.status {
	case 1:
		fmt.Println(res.meta)
		return input(u)
	case 2:
		mediaType, _, err := parseMeta(res.meta) // what to do with params
		if err != nil {
			fmt.Println("Unable to parse header meta\"", res.meta, "\":", err)
			return false
		}
		bodyBytes, err := ioutil.ReadAll(res.bodyReader)
		if err != nil {
			fmt.Println("Unable to read body.", err)
		}
		displayBody(bodyBytes, mediaType, *parsed) // does it need params?
	case 3:
		return urlHandler(res.meta) // TODO: max redirect times
	case 4, 5:
		fmt.Println(res.meta)
	case 6:
		fmt.Println("im not good enough in go to implement certs lol")
	}
	history = append(history, u)
	return true
}

func main() {
	//flag.ErrHelp = nil
	// command-line stuff
	flag.Usage = func() { // Usage override
		fmt.Fprintf(os.Stderr, "Usage: %s [FLAGS] [URL]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "For help on the TUI client, type ? at interactive prompt, or see gelim(1)\n")
	}
	flag.Parse()
	if *helpFlag { // Handling --help myself since pflag prints an ugly ErrHelp
		flag.Usage()
		return
	}

	u := flag.Arg(0) // URL
	if u != "" {
		if !strings.HasPrefix(u, "gemini://") {
			u = "gemini://" + u
			if *appendInput != "" {
				u = u + "?" + url.QueryEscape(*appendInput)
			}
			urlHandler(u)
		}
	}
	if *noInteractive {
		return
	}

	rl, err := readline.New("url/cmd, ? for help > ")
	if err != nil {
		panic(err)

	}
	defer rl.Close()

	for {
		cmd, err := rl.Readline()
		if err != nil {
			break
		}
		// Command dispatch
		switch strings.ToLower(cmd) {
		case "h", "help", "?":
			printHelp()
			continue
		case "":
			continue
		case "q", "x", "quit", "exit":
			os.Exit(0)
		case "r", "reload":
			urlHandler(history[len(history)-1])
		case "history":
			for i, v := range history {
				fmt.Println(i, v)
			}
		case "link", "l", "peek":
			fmt.Println("this will allow you to peek at the link")
			fmt.Println("TODO: handle args for command")
		case "b", "back":
			if len(history) < 2 {
				fmt.Println("nothing to go back to (try `history` to see history)")
				continue
			}
			u = history[len(history)-2]
			urlHandler(u)
			history = history[0 : len(history)-3]
		case "f", "forward":
			fmt.Println("todo :D")
		default:
			index, err := strconv.Atoi(cmd)
			if err != nil {
				// Treat this as a URL
				u = cmd
				if !strings.HasPrefix(u, "gemini://") {
					u = "gemini://" + u
				}
			} else {
				// link index lookup
				if len(links) < index {
					fmt.Println("invalid link index, I have", len(links), "links so far")
					continue
				}
				u = links[index-1]
			}
			urlHandler(u)
		}
	}
}
