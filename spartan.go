package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type SpartanResponse struct {
	status           int
	meta             string // not parsed into mediaType and params yet
	bodyReader       *bufio.Reader
	bodyReaderClosed bool
	conn             *net.Conn
	connClosed       bool
}

// SpartanParsedURL fetches u and resturns a SpartanResponse
func SpartanParsedURL(u *url.URL) (res *SpartanResponse, err error) {
	host := u.Host
	if u.Port() == "" {
		host += ":300"
	}
	// Connect to server
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return
	}
	// defer conn.Close()
	// Send request
	path := u.Path
	if u.Path == "" {
		path = "/"
	}
	input, err := url.QueryUnescape(u.RawQuery)
	if err != nil {
		return
	}
	conn.Write([]byte(fmt.Sprintf("%s %s %d\r\n", u.Hostname(), path, len(input))))
	conn.Write([]byte(input))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	header, err := reader.ReadString(byte('\n'))
	if err != nil {
		return
	}
	// Parse header
	statusParts := strings.SplitN(header, " ", 2)
	status, err := strconv.Atoi(statusParts[0])
	if err != nil {
		err = errors.New("Invalid response header")
		return
	}
	meta := strings.Trim(statusParts[1], "\r\n")
	res = &SpartanResponse{
		status:           status,
		meta:             meta,
		bodyReader:       reader,
		bodyReaderClosed: false, // idk
		conn:             &conn,
		connClosed:       false,
	}
	return
}
