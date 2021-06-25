gelim(1)

# NAME

gelim - a friendly line-mode gemini client


# SYNOPSIS

_gelim_ [OPTIONS] [URL]


# OPTIONS

\--help, -h
	get cli help/usage

\--input, -i _string_
	append "?" + _string_ percent-encoded to _URL_ (will be ignored if URL not provided)

\--no-interactive, -I
	don't enter the line-mode interface (useful for just wanting to view _URL_ then exit)

\--search , -s _string_
	search _string_ with the search engine. this takes priority over _URL_ and --input (which means _URL_ and --input will be ignored if this flag is used)


# INTERFACE

just run _gelim_ and optionally provide a url. it will start the line-mode interface.
at the prompt, you can directly enter a url, link index, or otherwise a command.

## COMMANDS

\help, h, ?
	get help for the interface

\quit, exit, q, x
	exit the program

\back, b
	go to previous url in history

\forward, f
	go to next url in history

\history
	print history

\reload, r
	reload current page

\links, link, peek, l _number_
	get link for link-index _number_ (what the link links to).
	if no _number_ specified, it prints a list of all the links in the current page

\url, current, cur, u
	print current url

\search, s _query_...
	search _query_ with search engine
