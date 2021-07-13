package main

import (
	"fmt"
	"strconv"
	"os"
	"strings"
)

type Command struct {
	aliases []string
	do      func(client *Client, args ...string)
	help    string
}


func printHelp() {
	maxWidth := 0
	var placeholder string
	curWidth := 0
	for name, cmd := range commands {
		placeholder = ""
		parts := strings.SplitN(cmd.help, ":", 2)
		if len(parts) == 2 {
			placeholder = strings.TrimSpace(parts[0])
		}
		curWidth = len(name) + 1 // 1 for space
		if placeholder!= "" {
			curWidth += len(placeholder) + 3 // <> and a space
		}
		if curWidth > maxWidth {
			maxWidth = curWidth
		}
	}
	minSepSpaceLen := 2 // min space between command and the description
	// Here comes the fun part
	// We are now *actually* printing the help
	fmt.Println("You can directy enter a url or link-index (number) at the prompt.")
	fmt.Println()
	fmt.Println("Otherwise, there are plenty of useful commands you can use.")
	fmt.Println("Arguments are separated by spaces, and quoting with ' and \" is supported like the shell, but escaping quotes is not support yet.")
	fmt.Println()
	fmt.Println("Commands:")
	var left string
	var spacesBetween int
	var desc string
	for name, cmd := range commands {
		placeholder = ""
		desc = ""
		parts := strings.SplitN(cmd.help, ":", 2)
		if len(parts) == 2 {
			placeholder = strings.TrimSpace(parts[0])
			desc = strings.TrimSpace(parts[1])
		}
		if placeholder != "" {
			left = fmt.Sprintf("%s <%s>", name, placeholder)
		} else {
			left = fmt.Sprintf("%s", name)
			desc = cmd.help
		}
		// TODO: wrap description with... aniswrap?
		// also maybe add some colors in the help!
		spacesBetween = maxWidth + minSepSpaceLen - len(left)
		fmt.Printf("  %s%s %s\n", left, strings.Repeat(" ", spacesBetween), desc)
	}
}

var commands = map[string]Command{
	"search": {
		aliases: []string{"s"},
		do: func(c *Client, args ...string) {
			c.Search(strings.Join(args, " "))
		},
		help: "query... : search with search engine",
	},
	"quit": {
		aliases: []string{"exit", "x", "q"},
		do: func(c *Client, args ...string) {
			os.Exit(0)
		},
		help: "exit gelim",
	},
	"reload": {
		aliases: []string{"r"},
		do: func(c *Client, args ...string) {
			if len(c.history) < 1 {
				fmt.Println(ErrorColor("No history yet!"))
				return
			}
			c.HandleParsedURL(c.history[len(c.history)-1])
		},
		help: "reload current page",
	},
	"history": {
		aliases: []string{"hist"},
		do: func(c *Client, args ...string) {
			for i, v := range c.history {
				fmt.Println(i, v.String())
			}
		},
		help: "print list of previously visited URLs",
	},
	"link": {
		aliases: []string{"l", "peek", "p", "links"},
		do: func(c *Client, args ...string) {
			if len(args) < 1 {
				for i, v := range c.links {
					fmt.Println(i+1, v)
				}
				return
			}
			var index int
			index, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(ErrorColor("invalid link index"))
				return
			}
			fmt.Println(c.GetLinkFromIndex(index))
		},
		help: "index : peek what a link index would link to. supply no arguments to see a list of current links",
	},
	"back": {
		aliases: []string{"b"},
		do: func(c *Client, args ...string) {
			if len(c.history) < 2 {
				fmt.Println(ErrorColor("nothing to go back to (try `history` to see history)"))
				return
			}
			c.HandleParsedURL(c.history[len(c.history)-2])
			c.history = c.history[0 : len(c.history)-2]
		},
		help: "go back in history",
	},
	"forward": {
		aliases: []string{"f"},
		do: func(c *Client, args ...string){
			fmt.Println("not implemented yet!")
		},
		help: "go forward in history",
	},
	"current": {
		aliases: []string{"u", "url", "cur"},
		do: func(c *Client, args ...string){
			if len(c.history) == 0 {
				fmt.Println("No history yet!")
				return
			}
			fmt.Println(c.history[0])
		},
		help: "print current url",
	},
}

