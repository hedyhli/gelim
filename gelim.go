package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"git.sr.ht/~adnano/go-xdg"
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
	versionFlag   = flag.BoolP("version", "v", false, "print the version and exit\n")
	configFlag    = flag.StringP("config", "c", "", "specify a different config location\n")
)

var (
	// quoteFieldRe greedily matches between matching pairs of '', "", or
	// non-word characters.
	quoteFieldRe = regexp.MustCompile("'(.*)'|\"(.*)\"|(\\S*)")
)

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
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

var (
	Version string = "version unknown"
)

func main() {
	// command-line stuff
	flag.Usage = func() { // Usage override
		fmt.Fprintf(os.Stderr, "Usage: %s [FLAGS] [URL]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "For help on the TUI client, type ? at interactive prompt, or see gelim(1)\n")
		fmt.Fprintf(os.Stderr, "For help on the TUI client, type ? at interactive prompt, or see gelim(1)\n")
	}
	flag.Parse()
	if *helpFlag { // Handling --help myself since pflag prints an ugly ErrHelp
		flag.Usage()
		return
	}

	if *versionFlag {
		fmt.Println("gelim", Version)
		return
	}

	configPath := filepath.Join(xdg.ConfigHome(), "gelim")
	if *configFlag != "" {
		_, err := os.Stat(*configFlag)
		if os.IsNotExist(err) {
			fmt.Printf("the specified config directory \"%s\" does not exist\n", *configFlag)
			os.Exit(1)
		} else if err != nil {
			fmt.Printf("unable to open the specified config directory \"%s\"\n", *configFlag)
			os.Exit(1)
		}
		configPath = *configFlag
	}

	u := ""
	cliURL := false // this is to avoid going to c.conf.StartURL if URL is visited from CLI

	c, err := NewClient(configPath)
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
			c.HandleURLWrapper(u)
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

	if !cliURL {
		if c.conf.StartURL != "" {
			c.HandleURLWrapper(c.conf.StartURL)
		} else {
			fmt.Println("Welcome! Use the 'help' command to get started.")
		}
	}

	// main loop
	for {
		var line string
		var err error

		color.Set(c.style.Prompt)
		promptLines := strings.Split(c.parsePrompt()+" ", "\n")
		for i, line := range promptLines {
			if i == len(promptLines)-1 {
				break
			}
			fmt.Println(line)
		}
		prompt := promptLines[len(promptLines)-1]
		rl := c.getLiner()
		if c.promptSuggestion != "" {
			line, err = rl.PromptWithSuggestion(prompt, c.promptSuggestion, -1)
			c.promptSuggestion = ""
		} else {
			line, err = rl.Prompt(prompt)
		}
		rl.Close()
		color.Unset()

		if err != nil {
			if err == ln.ErrPromptAborted || err == io.EOF {
				// Exit by ^C or ^D
				if err == io.EOF {
					fmt.Println("^D")
				}
				c.QuitClient(0)
			}
			c.style.ErrorMsg("Error reading input: " + err.Error())
			// Exiting because it will cause an infinite loop of error if used 'continue' here
			c.QuitClient(1)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Get our command and args! ✨
		cmd, cmdStr, args, ok := c.GetCommandAndArgs(line)
		// Vamos
		if ok {
			cmd.do(c, args...)
			continue
		}
		// Reaches here only if it was not a valid command
		if strings.Contains(cmdStr, ".") || strings.Contains(cmdStr, "/") {
			// looks like an URL

			var parsed *url.URL

			u = cmdStr
			parsed, err = url.Parse(u)
			if err != nil {
				c.style.ErrorMsg("Invalid url")
				continue
			}
			// Adding default scheme
			// Example:
			// If current url is example.com, and user would like to visit
			// example.com/foo.txt they can type "/foo.txt", and if they use
			// "foo.txt" it would lead to gemini://foo.txt which means if
			// current url is example.com/bar/ and user wants
			// example.com/bar/foo.txt, they can either use "/bar/foo.txt" or
			// "./foo.txt" so if user want to do relative path it has to start
			// with / or .
			//
			// TLDR
			// ----
			//   "foo.txt" -> "gemini://foo.txt"
			//   "./foo.txt" -> "gemini://current-url.org/foo.txt"
			if (parsed.Scheme == "" || parsed.Host == "") &&
				(!strings.HasPrefix(u, ".")) && (!strings.HasPrefix(u, "/")) {
				parsed, err = url.Parse("gemini://" + u)
				if err != nil {
					// Haven't actually encountered this case before (not
					// sure if it's even possible) but I'll put it here
					// just in case
					c.style.ErrorMsg("Invalid url")
					continue
				}
			}
			// this allows users to use relative urls at the prompt
			if len(c.history) != 0 {
				parsed = c.history[len(c.history)-1].ResolveReference(parsed)
			} else {
				if strings.HasPrefix(u, ".") || strings.HasPrefix(u, "/") {
					c.style.ErrorMsg("No history yet, cannot use relative URLs")
					continue
				}
			}
			c.HandleParsedURL(parsed)
			continue
		}
		// at this point the user input is probably not an url
		index, err := strconv.Atoi(cmdStr)
		if err != nil {
			// looks like an unknown command
			c.style.ErrorMsg("Unknown command. Hint: try typing ? and hit enter")
			continue
		}
		// link index lookup
		if len(c.history) == 0 {
			c.style.ErrorMsg("No history yet, cannot use link indexing")
			continue
		}
		u, isInput := c.GetLinkFromIndex(index)
		if u == "" {
			c.style.ErrorMsg("Empty URL for this input link!")
			continue
		}
		if isInput {
			c.Input(u, false)
			continue
		}
		c.HandleURLWrapper(u)
	}
}
