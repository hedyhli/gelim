package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/MarekStancik/readline"
	flag "github.com/spf13/pflag"
)

type Response struct {
	status    int
	meta      string
	bodyBytes []byte
}

var (
	links   []string = make([]string, 0, 100)
	history []string = make([]string, 0, 100)
)

var noInteractive = flag.BoolP("no-interactive", "I", false, "Don't go to the line-mode interface\n")
var appendInput = flag.StringP("input", "i", "", "Append input to url ('?' + percet-encode input)\n")

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
			link := currentURL.ResolveReference(parsedLink).String() // link url
			var label string                                         // link text
			if len(bits) == 1 {
				label = link
			} else {
				label = strings.Join(bits[1:], " ")
			}
			links = append(links, link)
			fmt.Printf("[%d] %s\n", len(links), label)
		} else {
			// This should really be wrapped, but there's
			// no easy support for this in Go's standard
			// library (says solderpunk)
			fmt.Println(line)
		}
	}
}

func connect(u url.URL) (res Response, err error) {
	// Connect to server
	conn, err := tls.Dial("tcp", u.Host+":1965", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println("Failed to connect: " + err.Error())
		return Response{}, nil
	}
	defer conn.Close()
	// Send request
	conn.Write([]byte(u.String() + "\r\n"))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	responseHeader, err := reader.ReadString('\n')
	parts := strings.Fields(responseHeader)
	status, err := strconv.Atoi(parts[0][0:1])
	meta := parts[1]
	bodyBytes, err := ioutil.ReadAll(reader)
	return Response{status, meta, bodyBytes}, err
}

// input handles input status codes
func input(u string) (ok bool) {
	stdinReader := bufio.NewReader(os.Stdin)
	fmt.Print("INPUT> ")
	query, _ := stdinReader.ReadString('\n')
	query = strings.TrimSpace(query)
	u = u + "?" + url.QueryEscape(query)
	return urlHandler(u)
}

// displayBody handles the displaying of body bytes for response
func displayBody(res Response, parsedURL url.URL) {
	// text/* content only
	if !strings.HasPrefix(res.meta, "text/") {
		fmt.Println("Unsupported type " + res.meta)
		return
	}
	body := string(res.bodyBytes)
	if res.meta == "text/gemini" {
		displayGeminiPage(body, parsedURL)
	} else {
		// Just print any other kind of text
		fmt.Print(body)
	}
}

func urlHandler(u string) bool {
	links = make([]string, 0, 100) // reset links
	// Parse URL
	parsed, err := url.Parse(u)
	if err != nil {
		fmt.Println("invalid url")
		return false
	}
	// connect and fetch
	res, err := connect(*parsed)
	if err != nil {
		fmt.Println(err)
		return false
	}
	// Switch on status code
	switch res.status {
	case 1:
		fmt.Println(res.meta)
		displayBody(res, *parsed)
		return input(u)
	case 2:
		displayBody(res, *parsed)
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
	// command-line stuff
	flag.Parse()

	u := flag.Arg(0) // URL
	if u != "" {
		if !strings.HasPrefix(u, "gemini://") {
			u = "gemini://" + u
		}
		if *appendInput != "" {
			u = u + "?" + url.QueryEscape(*appendInput)
		}
		urlHandler(u)
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
			history = history[0 : len(history)-2]
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
