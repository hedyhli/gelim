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
	"github.com/lmorg/readline"
	flag "github.com/spf13/pflag"
)

var (
	links     []string = make([]string, 0, 100)
	history   []string = make([]string, 0, 100)
	searchURL          = "gemini://geminispace.info/search" // TODO: make it configurable
)

var (
	promptColor      = color.New(color.FgCyan).SprintFunc()
	pagerPromptColor = color.New(color.FgMagenta).SprintFunc()
	ErrorColor       = color.New(color.FgRed).SprintFunc()
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

func printHelp() {
	fmt.Println("you can enter a url, link index, or a command.")
	fmt.Println()
	fmt.Println("commands")
	fmt.Println("  b           go back")
	fmt.Println("  q, x        quit")
	fmt.Println("  history     view history")
	fmt.Println("  r           reload")
	fmt.Println("  l <index>   peek at what a link would link to, supply no arguments to view all links")
	fmt.Println("  s <query>   search engine")
}

// Pager uses `less` to display body
// falls back to fmt.Print if errors encountered
func Pager(body string) {
	cmd := exec.Command("less", "-FSEXR", "--mouse", "-P "+pagerPromptColor("(PAGER) "))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		// yuck:
		fmt.Println("could not open a stdin pipe.", err)
		fmt.Println("not using pager")
		fmt.Println()
		fmt.Print(body)
		return
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		// yuck again
		fmt.Println("could not run pager.", err)
		fmt.Println("not using pager")
		fmt.Print(body)
		return
	}
	io.WriteString(stdin, body+"\n")
	stdin.Close()
	cmd.Stdin = os.Stdin
	cmd.Wait()
}

func getLinkFromIndex(i int) string {
	if len(links) < i {
		fmt.Println("invalid link index, I have", len(links), "links so far")
		return ""
	}
	return links[i-1]
}

func queryEscape(s string) string {
	return strings.Replace(url.QueryEscape(s), "+", "%20", -1)
}

func search(q string) {
	u := searchURL + "?" + queryEscape(q)
	GeminiURL(u)
}

func main() {
	//flag.ErrHelp = nil
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

	// TODO: idea - should the search use URL as search engine if URL arg present?
	// nah, should make the search engine configurable once the conf stuff is set up
	if *searchFlag != "" {
		search(*searchFlag) // it's "searchQuery" more like
	} else { // need else because when user use --search we should ignore URL and --input
		u = flag.Arg(0) // URL
		if u != "" {
			if *appendInput != "" {
				u = u + "?" + queryEscape(*appendInput)
			}
			GeminiURL(u)
		} else {
			// if --input used but url arg is not present
			if *appendInput != "" {
				fmt.Println("ERROR: --input used without an URL argument")
				// should we print usage?
				os.Exit(1)
			}
		}
	}
	if *noInteractive {
		return
	}

	// and now here comes the line-mode prompts and stuff

	rl := readline.NewInstance()
	rl.SetPrompt(promptColor("url/cmd, ? for help") + " > ")

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.CtrlC {
				os.Exit(0)
			}
			fmt.Println(ErrorColor("\nerror reading line input"))
			fmt.Println(ErrorColor(err))
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lineFields := QuotedFields(strings.ToLower(line))
		cmd := lineFields[0]
		var args []string
		if len(lineFields) > 1 {
			args = lineFields[1:]
		}
		// Command dispatch
		switch cmd {
		case "h", "help", "?":
			printHelp()
			continue
		case "q", "x", "quit", "exit":
			os.Exit(0)
		case "r", "reload":
			if len(history) < 1 {
				fmt.Println(ErrorColor("no history yet"))
				continue
			}
			GeminiURL(history[len(history)-1])
		case "history", "hist":
			for i, v := range history {
				fmt.Println(i, v)
			}
		case "link", "l", "peek", "links":
			if len(args) < 1 {
				for i, v := range links {
					fmt.Println(i+1, v)
				}
				continue
			}
			var index int
			index, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("invalid link index")
				continue
			}
			fmt.Println(getLinkFromIndex(index))
		case "b", "back":
			if len(history) < 2 {
				fmt.Println("nothing to go back to (try `history` to see history)")
				continue
			}
			u = history[len(history)-2]
			GeminiURL(u)
			history = history[0 : len(history)-3]
		case "f", "forward":
			fmt.Println("todo :D")
		case "s", "search":
			search(strings.Join(args, " "))
		default:
			if strings.Contains(cmd, ".") || strings.Contains(cmd, "/") {
				// look like an URL
				u = cmd
			} else {
				index, err := strconv.Atoi(cmd)
				if err != nil {
					// looks like an unknown command
					fmt.Println(ErrorColor("unknown command"))
					continue
				}
				// link index lookup
				u = getLinkFromIndex(index)
				if u == "" {
					continue
				}
			}
			GeminiURL(u)
		}
	}
}
