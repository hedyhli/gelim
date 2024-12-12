# Gelim

[![builds.sr.ht status](https://builds.sr.ht/~hedy/gelim.svg)](https://builds.sr.ht/~hedy/gelim)
[![Go Report Card](https://goreportcard.com/badge/git.sr.ht/~hedy/gelim)](https://goreportcard.com/report/git.sr.ht/~hedy/gelim)
[![libera.chat/#gelim](https://img.shields.io/badge/libera.chat-%23gelim-blue)](irc://irc.libera.chat/#gelim)

A minimalist line-mode smolnet client written in go.

![screenshot](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/gelim-pager.png)

[More screenshots](#screenshots)

You get a simple line-mode browser interface to save and navigate Gemini,
Spartan, and Nex URLs, plus a pager to view pages. Nothing else.

**Table of contents**

<!-- mtoc-start -->

* [Features](#features)
* [Install](#install)
  * [Prebuilt binaries](#prebuilt-binaries)
  * [With `go install`](#with-go-install)
  * [Build from source](#build-from-source)
  * [Troubleshooting](#troubleshooting)
* [Usage](#usage)
  * [Quickstart](#quickstart)
* [Navigating a page/document](#navigating-a-pagedocument)
* [Config](#config)
  * [Prompt format options](#prompt-format-options)
  * [Client certificates](#client-certificates)
* [A note about the pager](#a-note-about-the-pager)
  * [Mouse support](#mouse-support)
  * [Search in page](#search-in-page)
* [CLI Options](#cli-options)
* [Screenshots](#screenshots)
* [Behavior details](#behavior-details)
  * [Link indexing](#link-indexing)
  * [Redirects](#redirects)
  * [More...](#more)
  * [Inconsistent behavior?](#inconsistent-behavior)
* [Remotes](#remotes)
* [Bugs, features, feedback, and contributions](#bugs-features-feedback-and-contributions)
* [Development](#development)
* [Meta](#meta)
  * [Motivation](#motivation)

<!-- mtoc-end -->

## Features

- Searching from the command line
- Inputs from the command line
- Relative url at prompt
- Pager (requires less(1))
- Configuration
- [spartan:// protocol](gemini://spartan.mozz.us) support
- [nex:// protocol](https://nex.nightfall.city) support
- Copying to clipboard

## Install

### Prebuilt binaries

Download the latest binary from either SourceHut or GitHub:

- SourceHut: Visit https://git.sr.ht/~hedy/gelim and click on the "release
  notes" link on the latest version shown under "refs"
- GitHub: Visit [the latest release on
  GitHub](https://github.com/hedyhli/gelim/releases/latest) and scroll down to see
  the list of attached assets.

Prebuilt binaries for linux and darwin are provided with both x86 and ARM
architectures.

The tarball archive available for each release includes the **gelim** binary,
the license, this readme, and the `gelim(1)` manpage.

[Skip to usage](#usage)

### With `go install`

```sh
go install git.sr.ht/~hedy/gelim@latest
```

Note that this method does not provide version information

[Skip to usage](#usage)

### Build from source

First, install the dependencies:

* go (>=1.16)
* [scdoc](https://sr.ht/~sircmpwn/scdoc) (for building the manpage)

Clone the repo somewhere reasonable:

```sh
git clone https://git.sr.ht/~hedy/gelim
cd gelim
# Recommended to use latest pinned version, rather than @latest
# git checkout v0.0.0
```

Build, optionally set `PREFIX` (default is `PREFIX=/usr/local`):

```
make
```

If all goes well, install gelim.

Remember to use the **same** `PREFIX` too.

```sh
sudo make install

# Or without sudo:
# $ make PREFIX=~/.local install
```

If you don't want to build the manual, you may put your hacker hat on, dive into
the Makefile, and copy the `go build` command used to compile gelim. This lets
you store version info accessible at `--version`.

Or you know what? Just a `go build` works too! The resulting binary will be
in the current directory.

Optionally verify your installation:

```
make checkinstall
```

The gelim binary would be sitting at `$PREFIX/bin/` with manpage at
`$PREFIX/share/man/` :)


### Troubleshooting

* **Something to do with `io.ReadAll`**:

  Gelim requires go version >= 1.16.

If you're having other issues with installation, please send an email to the
[mailing list](mailto:~hedy/inbox@lists.sr.ht).


## Usage

Note that the `gelim(1)` manpage may not be the most recently updated. But new
features and things like that will definitely be put in there once it's tested
stable.

### Quickstart

```
gelim geminiprotocol.net
```
This will bring you to less(1). You can use less to browse the page normally.

*Note: if you see something like "-P is not an option", this is because your
system does not support one of gelim's default less options, you should skip
over to the 'config' section below, and configure your lessOpts to remove the -P
option, and any other your version of less doesn't have.*

When you want to visit a link, you have to quit less first. **Press `q`**

The page will be fetched and you'll be in less again.

Now let's try something more interesting.

While you're at the prompt type:
```
rawtext.club
```
Say you don't have an account on rawtext.club yet and would like to sign up.

Go to the bottom of the page, where the link to signing up is provided. **Type `G`**

Then, you have to quit the pager. **Press `q`**

Look for the link number that links to the sign up page, and enter it directly at the prompt.
As of writing, the link number is 38, but keep in mind this number may change when you are
trying this out.
```
38
```
And now you've decided to have a look at rawtext.club's values at the front page on more time.
Unfortunately, the sign up page does not provide a link to go back to home. No worries, you can
directly use the path (prefixed with . or /) at the prompt.

Let's try it out. Quit the pager (**`q`**), and **type `/` and press enter**

Voila, you're at the front page again!

Thanks for trying out this quickstart tutorial, there is still much to explore. Type in **`help`**
from the prompt and check out the commands, have fun!

## Navigating a page/document

Everything in page rendering is handled with less(1). less is called with
options specified from your config file. gelim has no intervention in
pre-processing any keys.

Hence, anything that works in less should work normally when less is called by
gelim.

Useful navigation keys include: `d`/`u`/`PgDn`/`PgUp`/`Space`.

Useful keys for jumping to positions include: `g`/`G`.

Use `q` to quit.

## Config

For people on a Unix system it will look for configuration in `~/.config/gelim/config.toml`.

Gelim follows XDG specification for deciding where to find config and where to
store data files.

Though you do not need a configuration file to have gelim working.

```toml
# example config

prompt = "-->"            # default: "%U\n>" (the full url of
                          # the current page), a new line and a ">".
                          # (more info below)

startURL = "example.com"  # default: ""
                          # the page to visit if starting gelim without
                          # a specified link

# will be put in LESS environment variable
lessOpts = "-FSXR~"       # default: "-FSXR~ -P pager (q to quit)"

searchURL = "gemini://kennedy.gemi.dev/search"

clipboardCopyCmd = "pbcopy"  # Example for MacOS. default = "" (unset)

maxRedirects = 5

index0shortcut = -1  # default: unset (0). an alias for link index 0

maxWidth = 90  # width of each page is max(<terminalWidth>, maxWidth)

useCertificates = [  # default: [] (see details below)
    "gemini://astrobotany.mozz.us",
    "gemini://bbs.geminispace.org",
]
```

**clipboardCopyCmd**:

Example for linux with xclip: `xclip -sel c`

Contents to be copied will be piped to the command as stdin.

**maxRedirects**:
- 0: Always confirm redirects
- `>0`: Ask to confirm redirects after a set number of redirects
- `<0`: Never confirm redirects. (Please see [this section](#redirects) for
  behavior details)

**index0shortcut**:

How gelim should treat link index argument "0"; please see [this
section](#link-indexing) for details.

**maxWidth**:

This value is only used for non-preformatted page formats. This includes
gemtext, where each line is a paragraph, but does not include `text/plain` and
Nex protocol directory pages.

### Prompt format options

You can use a number of placeholders for your prompt:

- `%U`: Full url of current page including scheme (gemini://example.com/foo/bar)
- `%u`: Full url of current page without scheme (example.com/foo/bar)
- `%P`: Absolute path of the current url (/foo/bar)
- `%p`: Base path of the current url (bar)
- `%H`: Hostname including port
- `%h`: Hostname without port

Use `%%` for a literal percent character, and percent-prefixed option that is not supported
will be ignored and presented literally.

The query part of the URL is stripped for all options.

Here are some examples:

```
config    resulting prompt
-------   -----------------
"%U>"     "gemini://example.com/foo/bar> "
"%P %%"   "/foo/bar % "
"%z>"     "%z> "
"%%%% $"  "%% $ "
```

Newline characters (`"\n"`) in the prompt are supported. The default prompt is
`%U\n>`, which produces:

```
gemini://example.com/foo/bar
> _
```

Where `_` indicates the cursor position.

### Client certificates

Save your `cert.pem` and `key.pem` files in the gelim config directory (such as
`~/.config/gelim/{cert,key}.pem`). Symlinks are supported.

Then use the `useCertificate` config option to specify the list of URL prefixes
that the client certificate should be used on. For example:

```toml
useCertificates = [
    "gemini://astrobotany.mozz.us/app",
    "gemini://bbs.geminispace.org",
]
```

If the current visiting URL has a prefix from this list, then the client
certificate, if available, will be used.

## A note about the pager

Gelim requires less(1) for paged output. If you don't have that installed, or is
on windows, it will print the page directly and you'll have to scroll the page
yourself (like AV-98). This is a bug and will be fixed in the near future.

### Mouse support

Add `--mouse` (if your version of less supports it) to `lessOpts` option
in your config file.

### Search in page

Type `/<your search query>` when less is running. See less(1) for more
information.


## CLI Options

```
~> gelim --help
Usage: ./gelim [FLAGS] [URL]

Flags:
  -c, --config string    specify a different config location
                         
  -h, --help             get help on the cli
  -i, --input string     append input to URL ('?' + percent-encoded input)
                         
  -I, --no-interactive   don't go to the line-mode interface
                         
  -s, --search string    search with the search engine (this takes priority over URL and --input)
                         
  -v, --version          print the version and exit
                         
For help on the TUI client, type ? at interactive prompt, or see gelim(1)
```

**Examples**:

Search
```
~> gelim -s "astrobotany"
```

Test your awesome, boring, classic, totally OG CGI scripts (exits immediately)
```
~> gelim example.org/cgi-bin/greet.sh -Ii 'world'
Hello, world!

~>
```

Use it in scripts to show outputs in stdout
```sh
~> cat <<EOF > python-help.sh
read -p "help> "
gelim do.hedy.dev/help/py -Ii "${REPLY:-help}"
EOF

~> bash python-help.sh
help> input
'Help on built-in function input in module builtins:

input(prompt=None, /)
    Read a string from standard input.  The trailing newline is stripped.

    The prompt string, if given, is printed to standard output without a
    trailing newline before reading input.

    If the user hits EOF (*nix: Ctrl-D, Windows: Ctrl-Z+Return), raise EOFError.
    On *nix systems, readline is used if available.'
```

(Single quotes added on output to aid syntax highlighting.)

Submit your twtxt to antenna
```
~> gelim -I 'gemini://warmedal.se/~antenna/submit' -i 'gemini://do.hedy.dev/tw.txt'
Thank you for your submission! Antenna has now been updated.
```


## Screenshots

**Commands**

![screenshot of `help` command output](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/gelim-cmds.png)

**Tour**

![screenshot of tour command](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/tour.png)

**Edit url**

![screenshot of editing url](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/editurl.png)

**Spartan**

![screenshot of spartan colors](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/spartan-colors.png)

**Links**

![screenshot of links](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/link.png)

**Search**

![screenshot of search command](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/search.png)

**Spartan input**

![screenshot of spartan input functionality](https://raw.githubusercontent.com/hedyhli/gelim/master/assets/spartan-input.png)


## Behavior details

A good way to master some software is to know how it works from the program's
perspective.

Let's take a look at some UX details from various parts of gelim that could
cause confusion.

- [Link indexing](#link-indexing)
- [Redirects](#redirects)


### Link indexing

Links on a page are numbered starting from 1. If a page has 3 links, the link
indices are 1, 2, and 3.

You can simply type 1, 2, or 3 at the line-mode prompt directly to visit them.

**Negative arguments**

-1 specifies the last link, -2 the second-last link and so on.

This pattern is applied for ALL commands that take in a link index as argument,
such as **link**, **tour**, **history**, **editurl**.

**Multiple arguments**

Commands **link** and **tour** supports multiple link index arguments, for
example:

```
> link 1 2
> tour 3 4 5
```

What they do with those links is dependent on what the command is for.

**Index 0**

This is undefined by default, but can act as a shortcut, set in config
`index0shortcut`.

A hint is printed when you do not set this config value, and you try to use a
`0` for link index:

```
> link 0
[ERROR] Behaviour for index 0 is undefined.
You can use -1 for accessing the last item, -2 for second last, etc.
Configure the behaviour of 0 in the config file.
Example: index0shortcut = -1, then whenever you use 0 it will be -1 instead.
This works for commands history, links, editurl, and tour.
```

**Ranges**

At the moment, ranges of link indices only work for command **tour**.

Ranges can be specified with an upper bound and a lower bound, separated by a
comma. Both bounds can be omitted.

The default value of upper bound is the first link index, and the default value
of lower bound is last link index.

Bounds are inclusive.

Naturally, if both bounds are excluded, all links on current page are selected.

```
gemini://gemini.circumlunar.space/> l
[...]
10 gemini://gemini.circumlunar.space/capcom/
11 gemini://rawtext.club:1965/~sloum/spacewalk.gmi
12 gemini://calcuode.com/gmisub-aggregate.gmi
13 gemini://caracolito.mooo.com/deriva/
14 gemini://gempaper.strangled.net/mirrorlist/
15 gemini://gemini.circumlunar.space/users/

gemini://gemini.circumlunar.space/> t 11,
Added 5 items to tour list

gemini://gemini.circumlunar.space/> t c
Cleared 5 items

gemini://gemini.circumlunar.space/> t ,
Added 15 items to tour list
```

### Redirects

The configuration option to set the maximum allowed automatic redirects is
`maxRedirects`. This is set to 5 by default (following RFC-2068), meaning gelim
will follow redirects 5 times, after which, if there are further redirects, user
will be prompted for what to do.

The command to view the history of redirects following current URL is
`redirects`.

Special values:
- **0**: Ask for input for all redirects
- **<0** (negative): Automatically follow all redirects

Due to implementation and resource limitations, gelim cannot practically follow
an infinite number of redirects, nor can gelim save the history of all previous
redirects and show them all in the `redirects` command.

The command shows a maximum of 10 most recent redirect URLs.

Once the number of unprompted redirects reaches 20, gelim aborts and prints the
10 most recent redirects.

If you wish, you can still copy the last URL from the output and visit the URL
as normal.

**What happens when the max number of redirects is reached**

Consider the example where `maxRedirects` is set to 2 in the configuration file.

This is the behavior of following the redirhell torture test URL:

```
[...]
So, with that said, the first link is to the "Redirection From Hell" test, a test
of a series of temporary redirects, always. The second link is to the next test.

 [1] Redirection From Hell
 [2] 0023
gemini://gemini.thebackupbox.net/test/torture/0022> 1
[WARNING] Max redirects of 2 reached
1 gemini://thebackupbox.net/test/redirhell/20190
2 gemini://thebackupbox.net/test/redirhell/25942

Redirect to:
gemini://thebackupbox.net/test/redirhell/26941
[y/n]> y
[WARNING] Max redirects of 2 reached
1 gemini://thebackupbox.net/test/redirhell/9114
2 gemini://thebackupbox.net/test/redirhell/4356

Redirect to:
gemini://thebackupbox.net/test/redirhell/2582
[y/n]>
```

After 2 redirects, the user will be prompted **and the count of redirects
resets**, meaning the user saying "y" to prompt is equivalent to the user
opening a new link of the said URL.

If the total number of times the user gets redirected for a particular website
is 10, and `maxRedirects` is set to 3, the user will be prompted 3 times on
whether to follow a subsequent redirect.

1. 3 redirects followed
1. Continue? yes
1. 3 redirects followed
1. Continue? yes
1. 3 redirects followed
1. Continue? yes
1. 1 redirect followed
1. the page loads and is displayed

Here is an example for the behavior of `redirects` command, following the case
where `maxRedirects` is set to 2:

```
[...]
So, with that said, the first link is to the "Redirection From Hell" test, a test
of a series of temporary redirects, always. The second link is to the next test.

 [1] Redirection From Hell
 [2] 0023
gemini://gemini.thebackupbox.net/test/torture/0022> 1
[WARNING] Max redirects of 2 reached
1 gemini://thebackupbox.net/test/redirhell/32043
2 gemini://thebackupbox.net/test/redirhell/1544

Redirect to:
gemini://thebackupbox.net/test/redirhell/22150
[y/n]> n
gemini://gemini.thebackupbox.net/test/torture/0022> redir
1 gemini://thebackupbox.net/test/redirhell/32043
2 gemini://thebackupbox.net/test/redirhell/1544
gemini://gemini.thebackupbox.net/test/torture/0022>
```

### More...

You can use `help <cmd>` at the prompt to view usage information for a
particular command. If there's anything that is confusing or a pattern/behavior
that is hard to remember, let me know!

Of course, a last resort could be studying the source code directly, remember to
look at the corresponding version which you built gelim from.

### Inconsistent behavior?

If there is any inconsistent behavior in gelim's interface, or if it does not
follow what is described in the docs, please let me know using a method of
contact described below.

## Remotes

- [SourceHut](https://sr.ht/~hedy/gelim)
- [Tildegit (gitea)](https://tildegit.org/hedy/gelim)
- [GitHub](https://github.com/hedyhli/gelim)
- [Codeberg](https://codeberg.org/hedy/gelim)

## Bugs, features, feedback, and contributions

**Questions and general feedback**:

* Send a ([plain text](https://useplaintext.email)) email to my
[public inbox](https://lists.sr.ht/~hedy/inbox).
* [How to subscribe to the mailing list without a sourcehut
  account](https://man.sr.ht/lists.sr.ht/#email-controls)
* Join `#gelim` on libera.chat IRC

**Bugs and feature requests**

* Submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim).
* You don't need a sourcehut account to subscribe or submit a ticket, [here's
  how to do everything with email](https://man.sr.ht/todo.sr.ht/#email-access)
* Or you can use the one on [github](https://github.com/hedyhli/gelim/issues) if
  you prefer.

**Pull requests, patches**

* Send patches to my [public inbox](https://lists.sr.ht/~hedy/inbox)
* If you prefer pull requests instead,
  [this](https://github.com/hedyhli/gelim/pulls) is where PRs should go. You
  could also send PRs to my public inbox but I'll have to search up how to
  merge them (lol)

## Development

```sh
make gelim
make gelim.1
# both
make all # or just make
make clean

make fmt
make checkfmt

# after tagging and pushing
# cp .envrc.example .envrc
# requires goreleaser, and sourcehut and github tokens
make release         # build and publish to GitHub (with goreleaser)
.scripts/release.sh  # uploads artifacts to sourcehut
```

## Meta

Gelim = "**ge**mini" + "**li**ne-**m**ode"-like interface

Pronunciation = Ge like "Jelly", lim like "limits"

### Motivation

once upon a time, a curious programmer stumbles upon `ssh
kiosk@gemini.circumlunar.space`. then tries out `bombadillo` and `amfora`...

"how do I move? most pagers allow pressing space... oops that opens the command
prompt"

"ok so I can press a number to go to that link... hmm how about 10, 11, 12 etc?
is it like vim where it has a timeout for numeric keys? oops, looks like only
single digits are supported..."

"wait so, single key press for single digit link indices, use a command prompt
for all others? okay sure"

Tries out `AV-98`

"I love the interface!"

"wait why is it scrolling to the bottom of the page already? like our good-ol
`cat`?"

"I have to manually scroll my terminal screen? I have to reach for my mouse?
otherwise I have to have a geeky window manager setup?"

"typing `go` command each time and navigating by relative URLs are a bit of a
pain"

...

"ok you know what: all links are commands. all link indices are commands. all
relative URL paths are commands. we shall put all content longer than screen's
height into a pager *everyone's familiar with*. [just like git (CLIs), but
without typing the `git`](https://git.sr.ht/~sircmpwn/shit)!"

the programmer sets off to code[...](https://yewtu.be/watch?v=dQw4w9WgXcQ)



<!--salutations, curious one.-->
