package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"mime"
	"strconv"

	//"errors"
	"net/url"
	"strings"

	"github.com/fatih/color"
)

type GeminiResponse struct {
	status           int
	meta             string // not parsed into mediaType and params yet
	bodyReader       *bufio.Reader
	bodyReaderClosed bool // I have no idea what I'm doing here
	conn             *tls.Conn
	connClosed       bool
}

// these should be used but atm it isnt, lol
//var (
//ErrConnFail       = errors.New("connection failed")
//ErrInvalidStatus  = errors.New("invalid status code")
//ErrDecodeMetaFail = errors.New("failed to decode meta header")
//)

var (
	h1Style   = color.New(color.Bold).Add(color.Underline).Add(color.FgYellow).SprintFunc()
	h2Style   = color.New(color.Bold).SprintFunc()
	linkStyle = color.New(color.FgBlue).SprintFunc()
	// style only applied to first line for some reason, so removing it all together :P
	// quoteStyle = color.New(color.Italic).SprintFunc()
)

// GeminiParsedURL fetches u and retruns *GeminiResponse
func GeminiParsedURL(u url.URL) (res *GeminiResponse, err error) {
	host := u.Host
	// Connect to server
	if u.Port() == "" {
		host += ":1965"
	}
	// TODO: cert cache and stuff
	conn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	// defer conn.Close()
	// Send request
	conn.Write([]byte(u.String() + "\r\n"))
	// Receive and parse response header
	reader := bufio.NewReader(conn)
	responseHeader, err := reader.ReadString('\n')
	// Parse header
	parts := strings.Fields(responseHeader)
	if len(parts) == 0 {
		conn.Close()
		return res, errors.New("Invalid response header: " + responseHeader)
	}
	status, err := strconv.Atoi(parts[0])
	if err != nil {
		conn.Close()
		return res, errors.New("Invalid status code")
	}
	meta := strings.Join(parts[1:], " ")
	res = &GeminiResponse{status, meta, reader, false, conn, false}
	return
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
