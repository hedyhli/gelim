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
)

type Response struct {
	status    int
	meta      string
	bodyBytes []byte
}

func printHelp() {
	fmt.Println("just enter a url")
	fmt.Println()
	fmt.Println("commands")
	fmt.Println("  b    go back")
	fmt.Println("  q    quit")
	fmt.Println("\nenter number to go to a link")
}

func displayGeminiPage(body string, currentURL url.URL) {
	links := make([]string, 0, 100)
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
			link := currentURL.ResolveReference(parsedLink).String()
			var label string
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
			// library
			fmt.Println(line)
		}
	}
}

func connect(u url.URL) (res Response) {
	// Connect to server
	conn, err := tls.Dial("tcp", u.Host+":1965", &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		fmt.Println("Failed to connect: " + err.Error())
		return Response{}
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
	return Response{status, meta, bodyBytes}
}

func main() {
	stdinReader := bufio.NewReader(os.Stdin)
	var u string // URL
	history := make([]string, 0, 100)
	for {
		fmt.Print("~> ")
		cmd, _ := stdinReader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)
		// Command dispatch
		switch strings.ToLower(cmd) {
		case "h", "help", "?":
			printHelp()
			continue
		case "":
			continue
		case "q":
			os.Exit(0)
		case "b":
			if len(history) < 2 {
				fmt.Println("No history yet!")
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
				//u = links[index-1]
				fmt.Println("link ", index-1)
			}
		}
		// Parse URL
		parsed, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL")
			continue
		}
		// connect and fetch
		res := connect(*parsed)
		// Switch on status code
		switch res.status {
		case 1:
			fmt.Println("imagine an input prompt here...")
		case 2:
			// Successful transaction
			// text/* content only
			if !strings.HasPrefix(res.meta, "text/") {
				fmt.Println("Unsupported type " + res.meta)
				continue
			}
			bodyBytes := res.bodyBytes
			if err != nil {
				fmt.Println("Error reading body")
				fmt.Println(err)
				continue
			}
			body := string(bodyBytes)
			if res.meta == "text/gemini" {
				displayGeminiPage(body, *parsed)
			} else {
				// Just print any other kind of text
				fmt.Print(body)
			}
			history = append(history, u)
		case 3:
			fmt.Println("imagine a redirect: " + res.meta)
		case 4, 5:
			fmt.Println("ERROR: " + res.meta)
		case 6:
			fmt.Println("im not good enough in go to implement certs lol")
		}
	}
}
