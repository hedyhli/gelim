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

var commands = map[string]Command{
	"help": {
		aliases: []string{"?", "h"},
		do: func(c *Client, args ...string) {
			// TODO: automatically generate help
			// TODO: command help (help <cmd>)
			fmt.Println(`
you can enter a url, link index, or a command.

commands
  b           go back
  q, x        quit
  history     view history
  r           reload
  l <index>   peek at what a link would link to, supply no arguments to view all links
  s <query>   search engine
  u, cur      print current url`)
		  },
		help: "help!",
	},
	"search": {
		aliases: []string{"s"},
		do: func(c *Client, args ...string) {
			c.Search(strings.Join(args, " "))
		},
		help: "search with search engine",
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
		help: "Print list of previously visited URLs",
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
		help: "peek what a link index would link to. supply no arguments to see a list of current links",
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

