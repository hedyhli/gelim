# Gelim

[![builds.sr.ht status](https://builds.sr.ht/~hedy/gelim.svg)](https://builds.sr.ht/~hedy/gelim)
[![Go Report Card](https://goreportcard.com/badge/git.sr.ht/~hedy/gelim)](https://goreportcard.com/report/git.sr.ht/~hedy/gelim)

A minimalist line-mode gemini client written in go.

**WARNING: the project is still in its early stages so do expect bugs and
incomplete features, if you encounter them or would like to suggest an
improvement, feel free to submit to the [ticket tracker](https://todo.sr.ht/~hedy/gelim)
on srht or the one on [github](https://github.com/hedyhli/gelim).**


![screenshot](https://hedy.smol.pub/gelim-pager.png)

[more screenshots](#screenshots)

You get a simple line-mode interface to navigate URLs, plus a pager to view
pages. Seriously, what more do you want?


## Features

- Searching from the command line
- Inputs from the command line
- Relative url at prompt
- Pager (requires less(1))
- Configuration
  - Custom search URL
  - Custom pager opts
  - and more
- [spartan:// protocol](gemini://spartan.mozz.us) support
- Copying to clipboard

## Install

I plan to set up automated uploads of pre-built binaries to each
[release](https://git.sr.ht/~hedy/gelim/refs) at some point in the future.
As of now the only option is to build from source.

First, install the dependencies:

* go (I think >=1.16)
* [scdoc](https://sr.ht/~sircmpwn/scdoc) (for building manpage)

Clone the repo somewhere reasonable:

```
git clone https://git.sr.ht/~hedy/gelim
cd gelim
# git checkout v0.0.0  # pin specific version or commit
```

Build, optionally set `PREFIX` (default is `PREFIX=/usr/local`):

```
make
```

If all goes well, install gelim:

```
make install
```

Remember to use the same `PREFIX` too.

Optionally verify your installation:

```
make checkinstall
```

The gelim binary would be sitting at `$PREFIX/bin/` with manpage at
`$PREFIX/share/man/` :)


### Troubleshooting

* **"scdoc: command not found"**:
  Make sure [scdoc](https://sr.ht/~sircmpwn/scdoc/) is installed before building.
* **Something to do with `io.ReadAll`**:
  Upgrade your go version to something higher or equal to v1.16 and then try again.

If you're having other issues with installation, please send an email to the
[mailing list](mailto:~hedy/inbox@lists.sr.ht) with errors/logs if available.


## Usage

If you used the Makefile to install gelim the manpage should automatically be
built and installed. See gelim(1)

I'm also planning to have a mirror of that manual hosted on man.sr.ht in the
future if easy access.

Note that the manpage may not be the most recently updated. But new features
and things like that will definitely be put in there once it's tested stable.

### Quickstart

```
gelim gemini.circumlunar.space
```
This will bring you to less(1). You can use less to browse the page normally.

*Note: if you see something like "-P is not an option", don't panic, this is
because your system does not support one of gelim's default less options, you
should skip over to the 'config' section below, and configure your lessOpts to
remove the mouse option, and any other your version of less doesn't have.*

When you want to visit a link, you have to quit less first. **Press `q`**

The page will be fetched and you'll be in less again.

Now let's try something more interesting.

While you're at the prompt type:
```
rawtext.club
```
Say you don't have an account on RTC yet and would like to sign up.

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
opptions specified from your config file. gelim has no intervention in
pre-processing any keys.

Hence, anything that works in less should work normally when less is called by
gelim.

Useful navigation keys include: `d`/`u`/`PgDn`/`PgUp`/`Space`.

Useful keys for jumping to positions include: `g`/`G`.

**In case you've never used a pager before, please keep calm under the
circumstance of realizing that ctrl-c/ctrl-d does not quit. Please press `q`
:P**


## Config

For people on a Unix system it will look for configuration in `~/.config/gelim/config.toml`.

Though you do not need a configuration file to have gelim working.

```toml
# example config

prompt = "-->"            # default: "%U" (the full url of
                          # the current page), more info
                          # below

startURL = "example.com"  # default: ""

# will be put in LESS environment variable
lessOpts = "-FSXR~"       # default: "-FSXR~ -P pager (q to quit)"

searchURL = "geminispace.info/search"  # this is the default

clipboardCopyCmd = "pbcopy"  # Example for MacOS. default = "" (unset)
```

**clipboardCopyCmd**: Example for linux with xclip: `xclip -sel c`

Contents to be copied will be piped to the command as stdin.


### Prompt format options

You can use a number of placeholders for your prompt (like PS1 in bash):

- `%U`: Full url of current page including scheme (gemini://example.com/foo/bar)
- `%u`: Full url of current page without scheme (example.com/foo/bar)
- `%P`: Absolute path of the current url (/foo/bar)
- `%p`: Base path of the current url (bar)

Use `%%` for a literal percent character, and percent-prefixed option that is not supported
will be ignored and presented literally.

The query part of the URL will be stripped for all options, for security reasons. (If the
input was to be sensitive -- 11 status code -- the full query percent-encoded would be
printed as the prompt, which could mean revealing passwords, etc. Hence the query including
the `?` is stripped.)

Here are some examples:

```
config    resulting prompt
-------   -----------------
"%U>"     "gemini://example.com/foo/bar> "
"%P %%"   "/foo/bar % "
"%z>"     "%z> "
"%%%% $"  "%% $ "
```

## A note about the pager

Gelim requires less(1) for paged output. If you don't have that installed, or is on windows,
it will print the page directly and you'll have to scroll the page yourself. This is a bug
and will be fixed in the near future.

### Mouse support

Add `--mouse` (if your version of less supports it) to `lessOpts` option
in your config file.

### Search in page

Type `/<your search query>` when less is running. See less(1) for more
information.

## Screenshots

**Commands**

![screenshot of `help` command output](https://hedy.smol.pub/gelim-cmds.png)

**Tour**

![screenshot of tour command](https://hedy.smol.pub/tour.png)

**Edit url**

![screenshot of editing url](https://hedy.smol.pub/editurl.png)

**Spartan**

![screenshot of spartan colors](https://hedy.smol.pub/spartan-colors.png)

**Links**

![screenshot of links](https://hedy.smol.pub/link.png)

**Search**

![screenshot of search command](https://hedy.smol.pub/search.png)

**Spartan input**

![screenshot of spartan input functionality](https://hedy.smol.pub/spartan-input.png)

## Remotes

- [SourceHut](https://sr.ht/~hedy/gelim)
- [Tildegit (gitea)](https://tildegit.org/hedy/gelim)
- [GitHub](https://github.com/hedyhli/gelim)
- [Codeberg](https://codeberg.org/hedy/gelim)

## Bugs, features, feedback, and contributions

**Questions and general feedback**:

* send a ([plain text](https://useplaintext.email)) email to my
[public inbox](https://lists.sr.ht/~hedy/inbox).
* [How to subscribe to the mailing list without a sourcehut
  account](https://man.sr.ht/lists.sr.ht/#email-controls)
* or join `#gelim` on libera.chat irc for questions and suggestions

**Bugs and feature requests**

* Submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim).
* you don't need a sourcehut account to subscribe or submit a ticket, [here's
  how to do them with email](https://man.sr.ht/todo.sr.ht/#email-access)
* Or you can also use the one on
  [github](https://github.com/hedyhli/gelim/issues) if you prefer.

**Pull request, patches**

* Send patches to my [public inbox](https://lists.sr.ht/~hedy/inbox)
* If you prefer pull requests instead,
  [this](https://github.com/hedyhli/gelim/pulls) is where PRs should go. You
  could also send PRs to my public inbox but I'll have to search up how to
  merge them (lol)


## Meta

Gelim = "**ge**mini" + "**li**ne-**m**ode"-like interface

Pronounciation = Ge like "Jelly", lim like "limits"

(Imagine the Ubuntu jellyfish learning calculus)

---

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
pain..."

---

"ok you know what: all links are commands. all link indices are commands. all
relative URL paths are commands. we shall put all content longer than screen's
height into a pager *everyone's familiar with*. [just like git (CLIs), but
without typing the `git`](https://git.sr.ht/~sircmpwn/shit)!"

the programmer sets off to code[...](https://yewtu.be/watch?v=dQw4w9WgXcQ)




<!--salutations, curious one.-->
