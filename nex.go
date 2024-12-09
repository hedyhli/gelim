// URL Handler for nex protocol: nex.nightfall.city

package main

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type NexResponse struct {
	bodyReader       *bufio.Reader
	bodyReaderClosed bool
	conn             *net.Conn
	connClosed       bool
	fileExt          string // From request
}

// NexParsedURL fetches u and returns a NexResponse
func NexParsedURL(u *url.URL) (res *NexResponse, err error) {
	host := u.Host
	if u.Port() == "" {
		host += ":1900" // Default port
	}
	// Connect to server, no TLS
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	path := u.Path
	if u.Path == "" {
		path = "/"
	}
	fileExt := "/"
	if !strings.HasSuffix(path, "/") {
		fileExt = strings.SplitN(path, ".", 2)[1]
	}
	// Requests are simply file paths
	conn.Write([]byte(fmt.Sprintf("%s\n", path)))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	res = &NexResponse{
		bodyReader:       reader,
		bodyReaderClosed: false, // idk
		conn:             &conn,
		connClosed:       false,
		fileExt:          fileExt,
	}
	return
}

func (c *Client) ParseNexDirectoryPage(page *Page) string {
	var linkStyle = c.style.gmiLink.Sprint
	body := string(page.bodyBytes)
	rendered := []string{}
	maxWidth := 0

	for _, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(line, "\r") {
			line = strings.Trim(line, "\r")
		}
		if strings.HasPrefix(line, "=>") {
			originalLine := line
			line = strings.TrimSpace(line[2:])
			if line == "" {
				// Empty link line
				rendered = append(rendered, originalLine)
				continue
			}
			bits := strings.Fields(line)
			parsedLink, err := url.Parse(bits[0])
			if err != nil {
				rendered = append(rendered, originalLine)
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
			linkLine += linkStyle(label)

			// TODO: Config for protocols that appends `(protocol-name)` at the
			// end of link
			if link.Scheme != "gemini" {
				linkLine += fmt.Sprintf(" (%s)", link.Scheme)
			}
			if len(c.links) < 10 {
				linkLine = " " + linkLine
			}
			rendered = append(rendered, linkLine)
		} else {
			// Normal paragraph
			rendered = append(rendered, line)
			if len(line) > maxWidth {
				maxWidth = len(line)
			}
		}
	}

	return c.Centered(rendered, maxWidth)
}
