gelim(1) "" "line-mode smolnet client"

# NAME

*gelim* - a minimalist gemini, spartan, and nex protocol client


# SYNOPSIS

*gelim* [ _OPTIONS_ ] [ _URL_ ]


# OPTIONS

*--help*, -h
	get cli help/usage

*--input*, -i _string_
	append "?" + _string_ percent-encoded to _URL_ (will be ignored if URL not provided)

*--no-interactive*, -I
	don't enter the line-mode interface (useful for just wanting to view _URL_ then exit)

*--search*, -s _string_
	search _string_ with the search engine. this takes priority over _URL_ and
	*--input* (which means _URL_ and *--input* will be ignored if this flag is used)

*--config*, -c _path_
	specify a non-standard config location (see *FILES*)


# INTERFACE

just run _gelim_ and optionally provide a url. it will start the line-mode interface.
at the prompt, you can directly enter a url, link index, or otherwise a command.

Commands are parsed by spaces, with support for shell-like quoting and escaping.

Gelim uses a lightweight readline-like library for parsing the command line
within the line-mode interface. By default, the prompt shows the current URL and
a ">" symbol, this can be configured (see *CONFIGURATION*).

At the prompt, common line editing keys are supported. Completion for gelim's
commands are also supported through *TAB* and *Shift+TAB*.

## COMMANDS

For details on each command, please consult the built-in help interface.

*help*, h, ? _[command]_
	get help for the interface

*quit*, exit, q, x
	exit the program

*back*, b
	go to previous url in history

*forward*, f
	go to next url in history

*history*, hist, his _[number]_
	print history, or visit an item in history

*reload*, r
	reload current page

*page*, p, view, print, display
	reload current page

*links*, link, peek, l _[number]_
	get link for link-index _number_ (what the link links to).
	if no _number_ specified, it prints a list of all the links in the current page

*url*, current, cur, u
	print current url

*editurl*, e, eu, edit
	edit and visit current url

*copyurl*, cu
	copy current url to clipboard using _clipboardCopyCmd_ (see
	_CONFIGURATION_).

*search*, s _query_...
	search _query_ with search engine

*redirects*, redir
	show redirects

*tour*, t, loop [ _go_ | _ls_ | _ranges or numbers_... ]
	save a list of URLs into a tour list. on each _tour_ invokation, visit the
	next item in tour.

*config* [ _e[dit]_ | _r[eload]_ ]
	edit or reload the currently active configuration.

# CONFIGURATION

An optional configuration file can be specified at
_$XDG_CONFIG_HOME/gelim/config.toml_.

Configuration should be written in TOML format. For example:

```
prompt = "-->"
startURL = "example.com"
searchURL = "gemini://kennedy.gemi.dev/search"
clipboardCopyCmd = "pbcopy"
maxRedirects = 5
useCertificates = [
    "gemini://astrobotany.mozz.us/app",
    "gemini://bbs.geminispace.org",
]
```

Keys are case-insensitive.

*startURL* = _URL_
	The URL to visit when starting gelim without arguments.
	Defaults to an empty string, which disables this feature.

*searchURL* = _URL_
	The query will be sent to this URL as _URL_?_query_ for the _search_ command
	and the _--search_ CLI flag.

	Default is "gemini://kennedy.gemi.dev/search".

*clipboardCopyCmd* = _STRING_
	Contents will be piped as stdin when the _copyurl_ (or other) command is
	used which copies something into the system clipboard.

	Defaults to an empty string, which disables this feature.

*maxRedirects* = _NUMBER_
	Control whether to ask for confirmation when a page redirects the client.
	
	- 0: Always confirm redirects
	- >0: Ask to confirm redirects after a set number of redirects
	- <0: Never confirm redirects

	This is _5_ by default, following the RFC-2068.

*maxWidth* = _NUMBER_
	Each page will be rendered to maximum of max(_maxWidth_, terminal width)
	columns wide for gemtext documents, and centered.

	Set to _0_ to always use the terminal width.

	Set to negative X to use a maxWidth of X but disable centering.

	For plain text documents, Nex directories, and gophermaps, the page will be
	centered based on the maximum width of the text in the document.

	Default is _70_.

*useCertificate* = _LIST_
	The list of full URL prefixes (including scheme) that should use the client
	certificate. The certificate and key files should be in the same directory
	as the default config file location. Symlinks are supported.

	See *CLIENT CERTIFICATES* for more information.

	By default, this is an empty list.

*prompt* = _STRING_
	Newlines are supported. The default prompt is "%U\n>". Here are the
	available components:

[[ *Name*
:[ *Replacement*
:< *Example*
|  %U
:  Full URL including scheme and query
:  gemini://example.org/file/path.gmi?query
|  %u
:  Full URL without scheme and query
:  example.org/file/path.gmi
|  %P
:  Full path
:  /file/path.gmi
|  %p
:  Basename in path
:  path.gmi
|  %H
:  Full hostname with port
:  x.example.org:1965
|  %h
:  Hostname without port
:  x.example.org

	
	Note that gopher type selectors are stripped for both _%P_ and _%p_.

# CLIENT CERTIFICATES

To use a client certificate on certain gemini sites for authentication, you
should prepare PEM-encoded files for the certificate and the key. Place them in
the gelim config directory (see *FILES*), named _cert.pem_ and _key.pem_
respectively. Symlinks are supported.

On startup, gelim will look for these two files in the config directory and
attempt to load these as a client certificate.

To use the client certificate, list the URLs that you want to use it on in the
*useCertificate* config option:

```
useCertificates = [
    "gemini://example.org",
]
```

In this example, or URLs that begins with "gemini://example.org" will use your
client certificate.

# FILES

The config directory _$XDG_CONFIG_HOME/gelim/_ is used by default. This is
usually _~/.config/gelim/_. A different directory can be specified using the
*--config* option.

The following files are read by gelim, relative to the config directory.

- config.toml (see *CONFIGURATION*)
- cert.pem
- key.pem

# SEE ALSO

A _README.md_ file should be included in your *gelim* installation. *gelim* also
includes a built-in help functionality accessible through the *help* command.

Up-to-date information and source code can also be found at
_https://github.com/hedyhli/gelim_.

# AUTHORS

Created and maintained by ~hedy <_hedy.dev@protonmail.com_>.
