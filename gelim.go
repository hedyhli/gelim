package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	ln "github.com/peterh/liner"
	flag "github.com/spf13/pflag"
)

// flags
var (
	noInteractive = flag.BoolP("no-interactive", "I", false, "don't go to the line-mode interface\n")
	appendInput   = flag.StringP("input", "i", "", "append input to URL ('?' + percent-encoded input)\n")
	helpFlag      = flag.BoolP("help", "h", false, "get help on the cli")
	searchFlag    = flag.StringP("search", "s", "", "search with the search engine (this takes priority over URL and --input)\n")
)

var (
	// quoteFieldRe greedily matches between matching pairs of '', "", or
	// non-word characters.
	quoteFieldRe = regexp.MustCompile("'(.*)'|\"(.*)\"|(\\S*)")
)

// QuotedFields is an alternative to strings.Fields (see:
// https://golang.org/pkg/strings#Fields) that respects spaces between matching
// pairs of quotation delimeters.
//
// For instance, the quoted fields of the string "foo bar 'baz etc'" would be:
//   []string{"foo", "bar", "baz etc"}
//
// Whereas the same argument given to strings.Fields, would return:
//   []string{"foo", "bar", "'baz", "etc'"}
func QuotedFields(s string) []string {
	submatches := quoteFieldRe.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(submatches))
	for _, matches := range submatches {
		// if a leading or trailing space is found, ignore that
		if matches[0] == "" {
			continue
		}
		// otherwise, find the first non-empty match (inside balanced
		// quotes, or a space-delimited string)
		var str string
		for _, m := range matches[1:] {
			if len(m) > 0 {
				str = m
				break
			}
		}
		out = append(out, str)
	}
	return out
}

// Pager uses `less` to display body
// falls back to fmt.Print if errors encountered
func Pager(body string, conf *Config) {
	cmd := exec.Command("less")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Print(body)
		return
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "LESS="+conf.LessOpts)
	if err := cmd.Start(); err != nil {
		fmt.Print(body)
		return
	}
	io.WriteString(stdin, body)
	stdin.Close()
	cmd.Stdin = os.Stdin
	cmd.Wait()
}

func queryEscape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func main() {
	// command-line stuff
	flag.Usage = func() { // Usage override
		fmt.Fprintf(os.Stderr, "Usage: %s [FLAGS] [URL]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "For help on the TUI client, type ? at interactive prompt, or see gelim(1)\n")
	}
	flag.Parse()
	if *helpFlag { // Handling --help myself since pflag prints an ugly ErrHelp
		flag.Usage()
		return
	}
	u := ""
	cliURL := false // this is to avoid going to c.conf.StartURL if URL is visited from CLI

	c, err := NewClient()
	if err != nil {
		c.style.ErrorMsg(err.Error())
		os.Exit(1)
	}
	if *searchFlag != "" {
		c.Search(*searchFlag) // it's "searchQuery" more like
		cliURL = true
	} else { // need else because when user use --search we should ignore URL and --input
		u = flag.Arg(0) // URL
		if u != "" {
			if *appendInput != "" {
				u = u + "?" + queryEscape(*appendInput)
			}
			c.HandleURL(u)
			cliURL = true
		} else {
			// if --input used but url arg is not present
			if *appendInput != "" {
				c.style.ErrorMsg("ERROR: --input used without an URL argument")
				// should we print usage?
				os.Exit(1)
			}
		}
	}
	if *noInteractive {
		return
	}

	if c.conf.StartURL != "" && !cliURL {
		c.HandleURL(c.conf.StartURL)
	}

	// and now here comes the line-mode prompts and stuff
	rl := c.mainReader

	for {
		var line string
		var err error

		color.Set(c.style.Prompt)
		if c.promptSuggestion != "" {
			line, err = rl.PromptWithSuggestion(c.parsePrompt()+" ", c.promptSuggestion, -1)
			c.promptSuggestion = ""
		} else {
			line, err = rl.Prompt(c.parsePrompt() + " ")
		}
		color.Unset()

		if err != nil {
			if err == ln.ErrPromptAborted {
				os.Exit(1)
			}
			c.style.ErrorMsg("\nerror reading line input: " + err.Error())
			os.Exit(1) // Exiting because it will cause an infinite loop of error if used 'continue' here
		}
		rl.AppendHistory(line)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// TODO: put arg splitting logic into client.go or cmd.go so it can respect Command.quotedArgs value
		lineFields := strings.Fields(line)
		cmd := strings.ToLower(lineFields[0])
		var args []string
		if len(lineFields) > 1 {
			args = lineFields[1:]
		}
		// Command stuff
		if ok := c.Command(cmd, args...); !ok {
			if strings.Contains(cmd, ".") || strings.Contains(cmd, "/") {
				// look like an URL
				u = cmd
				parsed, err := url.Parse(u)
				if err != nil {
					c.style.ErrorMsg("Invalid url")
					continue
				}
				// example:
				// if current url is example.com, and user would like to visit example.com/foo.txt
				// they can type "/foo.txt", and if they use "foo.txt" it would lead to gemini://foo.txt
				// which means if current url is example.com/bar/ and user wants example.com/bar/foo.txt,
				// they can either use "/bar/foo.txt" or "./foo.txt"
				// so if user want to do relative path it has to start with / or .
				if (parsed.Scheme == "" || parsed.Host == "") && (!strings.HasPrefix(u, ".")) && (!strings.HasPrefix(u, "/")) {
					parsed, err = url.Parse("gemini://" + u)
				}
				// this allows users to use relative urls at the prompt
				if len(c.history) != 0 {
					parsed = c.history[len(c.history)-1].ResolveReference(parsed)
				} else {
					if strings.HasPrefix(u, ".") && strings.HasPrefix(u, "/") {
						fmt.Println("no history yet, cannot use relative path")
					}
				}
				c.HandleParsedURL(parsed)
				continue
			}
			// at this point the user input is probably not an url
			index, err := strconv.Atoi(cmd)
			if err != nil {
				// looks like an unknown command
				c.style.ErrorMsg("Unknown command. Hint: try typing ? and hit enter")
				continue
			}
			// link index lookup
			u, spartanInput := c.GetLinkFromIndex(index)
			if u == "" {
				continue
			}
			if spartanInput {
				c.Input(u, false)
				continue
			}
			c.HandleURL(u)
		}
	}
}
