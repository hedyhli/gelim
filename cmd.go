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
		firstLine := strings.SplitN(cmd.help, "\n", 2)[0]
		parts := strings.SplitN(firstLine, ":", 2)
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
	fmt.Println("Arguments are separated by spaces, and quoting with ' and \" is supported\nlike the shell, but escaping quotes is not support yet.")
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
	firstLine := strings.SplitN(cmd.help, "\n", 2)[0]
	parts := strings.SplitN(firstLine, ":", 2)
	var placeholder, desc string
	if len(parts) == 2 {
		placeholder = strings.TrimSpace(parts[0])
		desc = strings.TrimSpace(parts[1])
	}
	left := ""
	if placeholder != "" {
		left = fmt.Sprintf("%s %s", name, placeholder)
	} else {
		left = name
		desc = firstLine
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

func (c *Client) ResolveNonPositiveIndex(index int, totalLength int) int {
	if index == 0 {
		if c.conf.Index0Shortcut == 0 {
			c.style.ErrorMsg("Behaviour for index 0 is undefined.")
			fmt.Println("You can use -1 for accessing the last item, -2 for second last, etc.")
			fmt.Println("Configure the behaviour of 0 in the config file. Example: index0shortcut = -1, then whenever you use 0 it will be -1 instead. This works for commands history, links, and tour.")
			return 0
		}
		index = c.conf.Index0Shortcut
	}
	if index < 0 {
		// Because the index is 1-indexed
		// if index is -1, the final index is totalLength
		index = totalLength + index + 1
	}
	return index
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
					extra := strings.SplitN(cmd.help, "\n", 2)[1]
					if extra != "" {
						fmt.Println()
						fmt.Println(extra)
					}
				}
				return
			}
			printHelp()
		},
		help: "<cmd...> : print the usage or the help for a command",
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
		help: "<cmd...> : see aliases for a command or all commands",
	},
}

var commands = map[string]Command{
	"search": {
		aliases: []string{"s"},
		do: func(c *Client, args ...string) {
			c.Search(strings.Join(args, " "))
		},
		help: "<query...> : search with search engine",
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
			if len(args) == 0 {
				for i, v := range c.history {
					fmt.Println(i+1, v.String())
				}
				return
			}
			// Ignores all other arguments
			index, err := strconv.Atoi(args[0])
			if err != nil {
				c.style.ErrorMsg("Invalid history index number. Could not convert to integer")
				return
			}
			if index = c.ResolveNonPositiveIndex(index, len(c.history)); index == 0 {
				return
			}
			if len(c.history) < index || index <= 0 {
				c.style.ErrorMsg(fmt.Sprintf("%d item(s) in history", len(c.history)))
				fmt.Println("Try `history` to view the history")
				return
			}
			// TODO: handle spartan input
			c.HandleParsedURL(c.history[index-1])
		},
		help: `<index> : print list of previously visited URLs, or visit an item in history
Examples:
  - history
  - history 1
  - history -3`,
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
			var err error
			for _, arg := range args {
				index, err = strconv.Atoi(arg)
				if err != nil {
					c.style.ErrorMsg(arg + ": Invalid link index")
					continue
				}
				index = c.ResolveNonPositiveIndex(index, len(c.links))
				if index == 0 {
					continue
				}
				if index < 1 || index > len(c.links) {
					c.style.ErrorMsg(arg + ": Invalid link index")
					continue
				}
				link, _ := c.GetLinkFromIndex(index)
				fmt.Println(index, link)  // TODO: also save the label in c.links
			}
		},
		help: `<index>... : peek what a link index would link to, or see the list of all links
You can use non-positive indexes too, see `+"`links 0`"+` for more information
Examples:
  - links
  - l 1
  - l -3
  - l 1 2 3`,
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
	"tour": {
		aliases: []string{"t", "loop"},
		do: func(c *Client, args ...string) {
			if len(args) == 0 { // Just `tour`
				if len(c.tourLinks) == 0 {
					c.style.ErrorMsg("Nothing to tour")
					return
				}
				if c.tourNext == len(c.tourLinks) {
					fmt.Println("End of tour :)")
					fmt.Println("Use `tour go 1` to go back to the beginning")
					return
				}
				c.HandleURL(c.tourLinks[c.tourNext])
				c.tourNext++
				return
			}
			// tour commands
			switch args[0] {
			case "ls", "l":
				current := ""
				for i, v := range c.tourLinks {
					current = ""
					if i == c.tourNext {
						current = " <--next"
					}
					fmt.Printf("%d %s%s\n", i+1, v, current)
				}
			case "clear", "c":
				fmt.Println("Cleared", len(c.tourLinks), "items")
				c.tourLinks = nil
				c.tourNext = 0
			case "go", "g":
				if len(args) == 1 {
					c.style.ErrorMsg("Argument expected for `go` subcommand.")
					fmt.Println("Use `tour ls` to list tour items, `tour go N` to go to the Nth item.")
					return
				}
				number, err := strconv.Atoi(args[1])
				if err != nil {
					c.style.ErrorMsg("Unable to convert " + args[1] + " to integer")
					return
				}
				if number = c.ResolveNonPositiveIndex(number, len(c.tourLinks)); number == 0 {
					return
				}
				if number > len(c.tourLinks) || number < 1 {
					c.style.ErrorMsg(fmt.Sprintf("%d item(s) in tour list", len(c.tourLinks)))
					fmt.Println("Use `tour ls` to list")
					return
				}
				// Because user provided number is 1-indexed and tourNext is 0-indexed
				c.HandleURL(c.tourLinks[number-1])
				c.tourNext = number
			case "*", "all":
				c.tourLinks = append(c.tourLinks, c.links...)
				fmt.Println("Added", len(c.links), "items to tour list")
			default: // `tour 1 2 3`, `tour 1,4 7 8 10,`
				if len(c.links) == 0 {
					c.style.ErrorMsg("No links yet")
					return
				}
				added := 0
				for _, v := range args {
					if strings.Contains(v, ",") {
						// start,end or start,
						// Without end will imply until the last link
						parts := strings.SplitN(v, ",", 2)
						if parts[1] == "" {
							// FIXME: avoid extra int->str->int conversion
							parts[1] = fmt.Sprint(len(c.links))
						}
						if parts[0] == "" {
							// FIXME: avoid extra int->str->int conversion
							parts[0] = "1"
						}
						start, err := strconv.Atoi(parts[0])
						end, err2 := strconv.Atoi(parts[1])

						if err != nil || err2 != nil {
							c.style.ErrorMsg("Number before or after ',' is not an integer: " + v)
							continue
						}
						if start > end {
							start, end = end, start
						}
						if start <= 0 || end > len(c.links) {
							c.style.ErrorMsg("Invalid range: " + v)
							continue
						}
						// start and end are both inclusive for us, but not for go
						c.tourLinks = append(c.tourLinks, c.links[start-1:end]...)
						added += len(c.links[start-1 : end])
						continue
					}
					// WIll reach here if it's not a range (no ',' in arg)
					number, err := strconv.Atoi(v)
					if err != nil {
						c.style.ErrorMsg("Unable to convert " + v + " to integer")
						continue
					}
					if number = c.ResolveNonPositiveIndex(number, len(c.links)); number == 0 {
						continue
					}
					if number > len(c.links) || number <= 0 {
						c.style.ErrorMsg(v + " is not in range of the number of links available")
						fmt.Println("Use `links` to see all the links")
						continue
					}
					c.tourLinks = append(c.tourLinks, c.links[number-1])
					added += 1
				}
				fmt.Println("Added", added, "items to tour list")
			}
		},
		help: `<range or number>... : loop over selection of links in current page
Use tour * to add all links. you can use ranges like 1,10 or 10,1 with single links as multiple arguments.
Use tour ls/clear to view items or clear all.
tour go <index> takes you to an item in the tour list
Examples:
  - tour ,5 6,7 -1 9 11,
  - tour ls
  - tour g 2
  - tour clear`,
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
	// 	help: "<key> <value>: set a configuration value for the current gelim session",
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
