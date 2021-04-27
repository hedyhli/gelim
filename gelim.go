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
	"regexp"
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
	links     []string = make([]string, 0, 100)
	history   []string = make([]string, 0, 100)
	searchURL          = "gemini://geminispace.info/search" // TODO: make it configurable
)

// flags
var (
	noInteractive = flag.BoolP("no-interactive", "I", false, "don't go to the line-mode interface\n")
	appendInput   = flag.StringP("input", "i", "", "append input to URL ('?' + percent-encoded input)\n")
	helpFlag      = flag.BoolP("help", "h", false, "get help on the cli")
	searchFlag    = flag.StringP("search", "s", "", "search with the search engine (this takes priority over URL and --input)\n")
)

var (
	// quoteFieldRe greedily matches between matching pairs of '', "", or
	// non-word characters.
	quoteFieldRe = regexp.MustCompile("'(.*)'|\"(.*)\"|(\\S*)")
)

// QuotedFields is an alternative to strings.Fields (see:
// https://golang.org/pkg/strings#Fields) that respects spaces between matching
// pairs of quotation delimeters.
//
// For instance, the quoted fields of the string "foo bar 'baz etc'" would be:
//   []string{"foo", "bar", "baz etc"}
//
// Whereas the same argument given to strings.Fields, would return:
//   []string{"foo", "bar", "'baz", "etc'"}
func QuotedFields(s string) []string {
	submatches := quoteFieldRe.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(submatches))
	for _, matches := range submatches {
		// if a leading or trailing space is found, ignore that
		if matches[0] == "" {
			continue
		}
		// otherwise, find the first non-empty match (inside balanced
		// quotes, or a space-delimited string)
		var str string
		for _, m := range matches[1:] {
			if len(m) > 0 {
				str = m
				break
			}
		}
		out = append(out, str)
	}
	return out
}

func printHelp() {
	fmt.Println("you can enter a url, link index, or a command.")
	fmt.Println()
	fmt.Println("commands")
	fmt.Println("  b           go back")
	fmt.Println("  q, x        quit")
	fmt.Println("  history     view history")
	fmt.Println("  r           reload")
	fmt.Println("  l <index>   peek at what a link would link to, supply no arguments to view all links")
	fmt.Println("  s <query>   search engine")
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
	u = u + "?" + queryEscape(query)
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
	status, err := strconv.Atoi(parts[0])
	if err != nil {
		fmt.Println("invalid status code:", parts[0])
		return false
	}
	statusGroup := status / 10
	meta := strings.Join(parts[1:], " ")
	res := Response{status, meta, reader}

	links = make([]string, 0, 100) // reset links

	switch statusGroup {
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
	default:
		fmt.Println("invalid status code:", res.status)
		return false
	}
	if (len(history) > 0) && (history[len(history)-1] != u) || len(history) == 0 {
		history = append(history, u)
	}
	return true
}

func getLinkFromIndex(i int) string {
	if len(links) < i {
		fmt.Println("invalid link index, I have", len(links), "links so far")
		return ""
	}
	return links[i-1]
}

func queryEscape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func search(q string) {
	u := searchURL + "?" + queryEscape(q)
	urlHandler(u)
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
	u := ""

	// TODO: idea - should the search use URL as search engine if URL arg present?
	// nah, should make the search engine configurable once the conf stuff is set up
	if *searchFlag != "" {
		search(*searchFlag) // it's "searchQuery" more like
	} else { // need else because when user use --search we should ignore URL and --input
		u = flag.Arg(0) // URL
		if u != "" {
			if !strings.HasPrefix(u, "gemini://") {
				u = "gemini://" + u
				if *appendInput != "" {
					u = u + "?" + queryEscape(*appendInput)
				}
				urlHandler(u)
			}
		} else {
			// if --input used but url arg is not present
			if *appendInput != "" {
				fmt.Println("ERROR: --input used without an URL argument")
				// should we print usage?
				os.Exit(1)
			}
		}
	}
	if *noInteractive {
		return
	}

	// and now here comes the line-mode prompts and stuff

	rl, err := readline.New("url/cmd, ? for help > ")
	if err != nil {
		panic(err)

	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineFields := QuotedFields(strings.ToLower(line))
		cmd := lineFields[0]
		var args []string
		if len(lineFields) > 1 {
			args = lineFields[1:]
		}
		// Command dispatch
		switch cmd {
		case "h", "help", "?":
			printHelp()
			continue
		case "q", "x", "quit", "exit":
			os.Exit(0)
		case "r", "reload":
			urlHandler(history[len(history)-1])
		case "history":
			for i, v := range history {
				fmt.Println(i, v)
			}
		case "link", "l", "peek", "links":
			if len(args) < 1 {
				for i, v := range links {
					fmt.Println(i+1, v)
				}
				continue
			}
			var index int
			index, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("invalid link index")
				continue
			}
			fmt.Println(getLinkFromIndex(index))
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
		case "s", "search":
			search(strings.Join(args, " "))
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
				u = getLinkFromIndex(index)
				if u == "" {
					continue
				}
			}
			urlHandler(u)
		}
	}
}
