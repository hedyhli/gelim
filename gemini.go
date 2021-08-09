package main

import (
	"mime"

	"github.com/fatih/color"
)

var (
	h1Style   = color.New(color.Bold).Add(color.Underline).Add(color.FgYellow).SprintFunc()
	h2Style   = color.New(color.Bold).SprintFunc()
	linkStyle = color.New(color.FgBlue).SprintFunc()
	// style only applied to first line for some reason, so removing it all together :P
	// quoteStyle = color.New(color.Italic).SprintFunc()
)

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
