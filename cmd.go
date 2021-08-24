package main

import (
	"fmt"
	"strconv"
	"strings"
)

type Command struct {
	aliases    []string
	do         func(client *Client, args ...string)
	help       string
	quotedArgs bool
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
		if placeholder != "" {
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
	fmt.Println("You can supply a command name to `help` to see the help for a specific command")
	fmt.Println()
	fmt.Println("Commands:")
	var spacesBetween int
	for name, cmd := range commands {
		// TODO: wrap description with... aniswrap?
		// also maybe add some colors in the help!
		parts := formatCommandHelp(&cmd, name, false)
		spacesBetween = maxWidth + minSepSpaceLen - len(parts[0])
		fmt.Printf("  %s%s %s\n", parts[0], strings.Repeat(" ", spacesBetween), parts[1])
	}
}

// Handles placeholders in cmd.help if any, if format is true it will return the placeholder
// string and the help string concatenated, if format is false, it returns them separately.
func formatCommandHelp(cmd *Command, name string, format bool) (formatted []string) {
	parts := strings.SplitN(cmd.help, ":", 2)
	var placeholder, desc string
	if len(parts) == 2 {
		placeholder = strings.TrimSpace(parts[0])
		desc = strings.TrimSpace(parts[1])
	}
	left := ""
	if placeholder != "" {
		left = fmt.Sprintf("%s <%s>", name, placeholder)
	} else {
		left = name
		desc = cmd.help
	}
	formatted = make([]string, 2)
	if format {
		formatted[0] = fmt.Sprintf("%s  %s", left, desc)
		return
	}
	formatted[0] = left
	formatted[1] = desc
	return
}

// Commands that reference variable commands, putting them separtely to avoid
// initialization cycle
var metaCommands = map[string]Command{
	"help": {
		aliases: []string{"h", "?", "hi"},
		do: func(c *Client, args ...string) {
			if len(args) > 0 {
				for _, v := range args {
					// Yes, have to do metaCommands manually
					switch v {
					case "help", "?", "h", "hi":
						fmt.Println("You literally just get help :P")
						return
					case "alias", "aliases", "synonymn":
						fmt.Println("See aliases for a command or all commands")
						return
					}

					cmd, ok := c.LookupCommand(v)
					if !ok {
						fmt.Println(v, "command not found")
						return
					}
					formatted := formatCommandHelp(&cmd, v, true)
					fmt.Println(formatted[0])
					// Extra help for command if the command supports it
					// c.Command(v, "help")
					// if len(args) != 1 {
					// 	fmt.Println()
					// }
				}
				return
			}
			printHelp()
		},
		help: "cmd : print the usage or the help for a command",
	},
	"aliases": {
		aliases: []string{"alias", "synonym"},
		do: func(c *Client, args ...string) {
			if len(args) > 0 {
				for _, v := range args {
					// I'm so tired having to do this stupid switch again and again for metaCommands
					// but I can't find a better solution UGH
					switch v {
					case "help", "?", "h", "hi":
						fmt.Println("help, ?, h, hi")
						return
					case "alias", "aliases", "synonym":
						fmt.Println("alias, aliases, synonym")
						return
					}
					cmd, ok := c.LookupCommand(v)
					if !ok {
						fmt.Println(v, "command not found")
					}
					fmt.Println(strings.Join(cmd.aliases, ", "))
					return
				}
			}
			fmt.Println("todo")
		},
		help: "cmd : see aliases for a command or all commands",
	},
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
			c.QuitClient()
		},
		help: "exit gelim",
	},
	"reload": {
		aliases: []string{"r"},
		do: func(c *Client, args ...string) {
			if len(c.history) < 1 {
				c.style.ErrorMsg("No history yet!")
				return
			}
			c.HandleParsedURL(c.history[len(c.history)-1])
		},
		help: "reload current page",
	},
	"history": {
		aliases: []string{"hist"},
		do: func(c *Client, args ...string) {
			// TODO: go to an url in history
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
				c.style.ErrorMsg("invalid link index")
				return
			}
			link, _ := c.GetLinkFromIndex(index)
			fmt.Println(link)
		},
		help: "index : peek what a link index would link to. supply no arguments to see a list of current links",
	},
	"back": {
		aliases: []string{"b"},
		do: func(c *Client, args ...string) {
			if len(c.history) < 2 {
				c.style.ErrorMsg("nothing to go back to (try `history` to see history)")
				return
			}
			c.HandleParsedURL(c.history[len(c.history)-2])
			c.history = c.history[0 : len(c.history)-2]
		},
		help: "go back in history",
	},
	"forward": {
		aliases: []string{"f"},
		do: func(c *Client, args ...string) {
			fmt.Println("not implemented yet!")
		},
		help: "go forward in history",
	},
	"current": {
		aliases: []string{"u", "url", "cur"},
		do: func(c *Client, args ...string) {
			if len(c.history) == 0 {
				fmt.Println("No history yet!")
				return
			}
			fmt.Println(c.history[len(c.history)-1])
		},
		help: "print current url",
	},
	"editurl": {
		aliases: []string{"e", "edit"},
		do: func(c *Client, args ...string) {
			// TODO: Use a link from current page or from history instead of current url
			if len(c.history) != 0 {
				c.promptSuggestion = c.history[len(c.history)-1].String()
			}
		},
		help: "edit the current url",
	},
	// TODO: didn't have time to finish this lol
	// "config": {
	// 	aliases: []string{"c", "conf"},
	// 	do: func(c *Client, args ...string) {
	// 		field := reflect.ValueOf(c.conf).Elem().FieldByName(args[0])
	// 		// if field == 0 {
	// 		// 	fmt.Println("key", args[0], "not found")
	// 		// 	return
	// 		// }
	// 		field.Set(reflect.Value{args[1]})
	// 		return
	// 	},
	// 	help: "key <space> value : set a configuration value for the current gelim session",
	// 	quotedArgs: true,
	// },
}

func CommandCompleter(line string) (c []string) {
	for name := range commands {
		if strings.HasPrefix(name, strings.ToLower(line)) {
			c = append(c, name)
		}
	}
	return
}
