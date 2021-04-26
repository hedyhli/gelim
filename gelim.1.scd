gelim(1)

# NAME

gelim - a friendly line-mode gemini client


# SYNOPSIS

_gelim_ [FLAGS] [URL]


# OPTIONS

\--help, -h
	get cli help/usage

\--input, -i _string_
	append "?" + _string_ percent-encoded to _URL_ (will be ignored if URL not provided)

\--no-interactive, -I
	don't enter the line-mode interface (useful for just wanting to view _URL_ then exit)

# INTERFACE

just run _gelim_ and optionally provide a url. it will start the line-mode interface.
at the prompt, you can directly enter a url, link index, or otherwise a command.

## COMMANDS

\help, h, ?
	get help for the interface

\back, b
	go to previous url in history

\forward, f
	go to next url in history

\history
	print history
