package main

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type GopherResponse struct {
	bodyReader *bufio.Reader
	conn       *net.Conn
	connClosed bool
	gophertype string
}

var gophertypes = map[string]string{
	"0": "TXT",
	"1": "DIR",
	"3": "ERR",
	"4": "BIN",
	"5": "DOS",
	"6": "UUE",
	"7": "SEARCH",
	"8": "TEL",
	"9": "BIN",
	"g": "GIF",
	"G": "GMI",
	"h": "HTML",
	"I": "IMG",
	"p": "PNG",
	"s": "SND",
	"S": "SSH",
	"T": "TEL",
}

// GopherParsedURL fetches u and returns a GopherResponse
func GopherParsedURL(u *url.URL) (res *GopherResponse, err error) {
	host := u.Host
	if u.Port() == "" {
		host += ":70"
	}
	// Connect to server, no TLS
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	fullpath := strings.TrimPrefix(u.Path, "/")
	if fullpath == "" || fullpath == "1" {
		fullpath = "1/"
	}
	pathParts := strings.SplitN(fullpath, "/", 2)
	gophertype := pathParts[0]

	fmt.Fprintf(conn, "%s\n", pathParts[1])
	reader := bufio.NewReader(conn)
	res = &GopherResponse{
		bodyReader: reader,
		conn:       &conn,
		connClosed: false,
		gophertype: gophertype,
	}
	return
}

func (c *Client) ParseGophermap(page *Page) string {
	var linkStyle = c.style.gmiLink.Sprint
	body := string(page.bodyBytes)
	rendered := []string{}
	dedents := []int{}

	for _, line := range strings.Split(body, "\n") {
		line = strings.Trim(line, "\r\n")
		if line == "." {
			rendered = append(rendered, "")
			dedents = append(dedents, 0)
			continue
		}

		columns := strings.Split(line, "\t")
		var title string
		if len(columns[0]) > 1 {
			title = columns[0][1:]
		} else if len(columns[0]) == 1 {
			title = ""
		} else {
			title = ""
			columns[0] = "i"
		}

		if len(columns) < 4 || strings.HasPrefix(columns[0], "i") {
			dedents = append(dedents, 0)
			rendered = append(rendered, title)
		} else {
			host := columns[2]
			port := columns[3]
			gtype := string(columns[0][0])
			path := columns[1]

			link := fmt.Sprintf("gopher://%s:%s/%s%s", host, port, gtype, path)
			switch gtype {
			case "8", "T":
				link = fmt.Sprintf("telnet://%s:%s", host, port)
			case "G":
				link = fmt.Sprintf("gemini://%s:%s%s", host, port, path)
			case "h":
				u, tf := isWebLink(path)
				if tf {
					if strings.Index(u, "://") > 0 {
						link = u
					} else {
						link = fmt.Sprintf("http://%s", u)
					}
				} else {
					link = fmt.Sprintf("gopher://%s:%s/h%s", host, port, path)
				}
			case "7":
				c.inputLinks = append(c.inputLinks, len(c.links))
			}
			c.links = append(c.links, link)
			gophertype := "(" + getGophertype(string(columns[0][0])) + ")"
			linkLine := fmt.Sprintf("%s  [%d] %s", gophertype, len(c.links), linkStyle(title))
			dedents = append(dedents, len(gophertype)+2)
			rendered = append(rendered, linkLine)
		}
	}

	return c.Centered(rendered, 0, dedents)
}

func isWebLink(resource string) (string, bool) {
	split := strings.SplitN(resource, ":", 2)
	if first := strings.ToUpper(split[0]); first == "URL" && len(split) > 1 {
		return split[1], true
	}
	return "", false
}

func getGophertype(t string) string {
	if val, ok := gophertypes[t]; ok {
		return val
	}
	return "???"
}
