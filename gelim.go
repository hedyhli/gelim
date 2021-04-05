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

func main() {
	stdinReader := bufio.NewReader(os.Stdin)
	var u string // URL
	links := make([]string, 0, 100)
	history := make([]string, 0, 100)
	for {
		fmt.Print("> ")
		cmd, _ := stdinReader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)
		// Command dispatch
		switch strings.ToLower(cmd) {
		case "": // Nothing
			continue
		case "q": // Quit
			fmt.Println("Bye!")
			os.Exit(0)
		case "b": // Back
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
				// Treat this as a menu lookup
				u = links[index-1]
			}
		}
		// Parse URL
		parsed, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL!")
			continue
		}
		// Connect to server
		conn, err := tls.Dial("tcp", parsed.Host+":1965", &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			fmt.Println("Failed to connect: " + err.Error())
			continue
		}
		defer conn.Close()
		// Send request
		conn.Write([]byte(u + "\r\n"))
		// Receive and parse response header
		reader := bufio.NewReader(conn)
		responseHeader, err := reader.ReadString('\n')
		parts := strings.Fields(responseHeader)
		status, err := strconv.Atoi(parts[0][0:1])
		meta := parts[1]
		// Switch on status code
		switch status {
		case 1, 3, 6:
			// No input, redirects or client certs
			fmt.Println("Unsupported feature!")
		case 2:
			// Successful transaction
			// text/* content only
			if !strings.HasPrefix(meta, "text/") {
				fmt.Println("Unsupported type " + meta)
				continue
			}
			// Read everything
			bodyBytes, err := ioutil.ReadAll(reader)
			if err != nil {
				fmt.Println("Error reading body")
				continue
			}
			body := string(bodyBytes)
			if meta == "text/gemini" {
				// Handle Gemini map
				links = make([]string, 0, 100)
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
						link := parsed.ResolveReference(parsedLink).String()
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
			} else {
				// Just print any other kind of text
				fmt.Print(body)
			}
			history = append(history, u)
		case 4, 5:
			fmt.Println("ERROR: " + meta)
		}
	}
}
