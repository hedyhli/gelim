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
		host += ":1900"
	}
	// Connect to server
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	// Send request
	path := u.Path
	if u.Path == "" {
		path = "/"
	}
	fileExt := "/"
	if !strings.HasSuffix(path, "/") {
		fileExt = strings.SplitN(path, ".", 2)[1]
	}
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

	// termWidth, _, err := term.GetSize(0)
	// if err != nil {
	// 	// TODO do something
	// 	c.style.ErrorMsg("Error getting terminal size")
	// 	return ""
	// }
	// sides := int(float32(termWidth) * c.conf.LeftMargin)
	// width := termWidth - sides
	// if width > c.conf.MaxWidth {
	// 	width = c.conf.MaxWidth
	// }

	rendered := ""
	body := string(page.bodyBytes)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasSuffix(line, "\r") {
			line = strings.Trim(line, "\r")

		} else if strings.HasPrefix(line, "=>") {
			originalLine := line
			line = strings.TrimSpace(line[2:])
			if line == "" {
				// Empty link line
				rendered += originalLine + "\n"
				continue
			}
			bits := strings.Fields(line)
			parsedLink, err := url.Parse(bits[0])
			if err != nil {
				rendered += originalLine + "\n"
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
			rendered += linkLine + "\n"
		} else {
			// Normal paragraph
			// rendered += ansiwrap.GreedyIndent(line, width, sides, sides) + "\n"
			rendered += line + "\n"
		}
	}
	// Remove last \n
	if len(rendered) > 0 {
		rendered = rendered[:len(rendered)-1]
	}
	return rendered
}
