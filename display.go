package main

import (
	"fmt"

	"github.com/fatih/color"
)

type Style struct {
	Error       *color.Color
	Warning     *color.Color
	Prompt      color.Attribute
	StatusError *color.Color // Mostly 5x, 6x, 4x

	// Gemtext rendering
	gmiH1    *color.Color
	gmiH2    *color.Color
	gmiH3    *color.Color
	gmiLink  *color.Color
	gmiQuote *color.Color
	gmiPre   *color.Color

	// Line mode interface
	cmdSynopsis    *color.Color
	cmdPlaceholder *color.Color
	cmdLabels      *color.Color // Eg: Usage:
}

var DefaultStyle = Style{
	Error:       color.New(color.FgRed),
	Warning:     color.New(color.FgYellow),
	Prompt:      color.FgCyan,
	StatusError: color.New(color.FgYellow),

	gmiH1:    color.New(color.Bold, color.Underline, color.FgYellow),
	gmiH2:    color.New(color.Bold, color.FgMagenta),
	gmiH3:    color.New(color.FgHiGreen),
	gmiPre:   color.New(color.FgYellow),
	gmiLink:  color.New(color.FgBlue),
	gmiQuote: color.New(color.Italic, color.FgGreen),

	cmdSynopsis:    color.New(color.Italic),
	cmdPlaceholder: color.New(color.FgBlue, color.Italic),
	cmdLabels:      color.New(color.Bold),
}

var (
	colorError   = color.FgRed
	colorWarning = color.FgYellow
	colorPrompt  = color.FgBlue
)

// StyleSprint returns msg with color, if color is nil or is a nil pointer, it returns msg untouched
func (s *Style) StyleSprint(color *color.Color, msg string) string {
	if color == nil {
		return msg
	}
	return color.Sprint(msg)
}

// ErrorMsg displays a formatted and colored message for msg, colored in Style.Error color
func (s *Style) ErrorMsg(msg string) {
	fmt.Printf("[%s] %s\n", s.StyleSprint(s.Error, "ERROR"), msg)
}

// WarningMsg displays a formatted and colored message for msg, colored in Style.Warning color
func (s *Style) WarningMsg(msg string) {
	fmt.Printf("[%s] %s\n", s.StyleSprint(s.Warning, "WARNING"), msg)
}

// PrintStatus takes the status code and the message and prints a colored message
func (s *Style) PrintStatus(code int, msg string) {
	// TODO: have a map or something so we can lookup what code is display
	// the default msg (like "not found"), and then append servers custom msg
}
