package main

import (
	"github.com/fatih/color"
)

var (
	h1Style   = color.New(color.Bold).Add(color.Underline).Add(color.FgYellow).SprintFunc()
	h2Style   = color.New(color.Bold).SprintFunc()
	linkStyle = color.New(color.FgBlue).SprintFunc()
	// style only applied to first line for some reason, so removing it all together :P
	// quoteStyle = color.New(color.Italic).SprintFunc()
)
