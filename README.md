# gelim

a minimalist line-mode gemini client written in go.

**WARNING: the project is still in its early stages so do expect bugs and
incomplete features, if you encounter them or would like to suggest an
improvement, feel free to submit to the [ticket tracker](https://todo.sr.ht/~hedy/gelim)
on srht or the one on [github](https://github.com/hedyhli/gelim).**


![screenshot](https://hedy.smol.pub/gelim-pager.png)

[more screenshots](#screenshots)


## features

- searching from the command line
- inputs from the command line
- relative url at prompt
- pager (requires less(1))
- configuration file
  - custom search URL
  - custom pager opts
  - and more
- [spartan:// protocol](gemini://spartan.mozz.us) support

## install

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


### troubleshooting

* **"scdoc: command not found"**:
  Make sure [scdoc](https://sr.ht/~sircmpwn/scdoc/) is installed before building.
* **Something to do with `io.ReadAll`**:
  Upgrade your go version to something higher or equal to v1.16 and then try again.

If you're having other issues with installation, please send an email to the
[mailing list](mailto:~hedy/inbox@lists.sr.ht) with errors/logs if available.


## usage

If you used the Makefile to install gelim the manpage should automatically be
built and installed. See gelim(1)

I'm also planning to have a mirror of that manual hosted on man.sr.ht in the
future if easy access.

Note that the manpage may not be the most recently updated. But new features
and things like that will definitely be put in there once it's tested stable.

### quickstart

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

## config

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
```

### prompt options

you can use a number of options for your prompt (like PS1 in bash):

- `%U`: full url of current page including scheme (gemini://example.com/foo/bar)
- `%u`: full url of current page without scheme (example.com/foo/bar)
- `%P`: absolute path of the current url (/foo/bar)
- `%p`: base path of the current url (bar)

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

## a note about the pager

gelim requires less(1) for paged output. If you don't have that installed, or is on windows,
it will print the page directly and you'll have to scroll the page yourself. This is a bug
and will be fixed in the near future.

### mouse support

Add `--mouse` (if your version of less supports it) to `lessOpts` option
in your config file.

## screenshots

**commands**

![screenshot of `help` command output](https://hedy.smol.pub/gelim-cmds.png)

**tour**

![screenshot of tour command](https://hedy.smol.pub/tour.png)

**edit url**

![screenshot of editing url](https://hedy.smol.pub/editurl.png)

**spartan**

![screenshot of spartan colors](https://hedy.smol.pub/spartan-colors.png)

**links**

![screenshot of links](https://hedy.smol.pub/link.png)

**search**

![screenshot of search command](https://hedy.smol.pub/search.png)

**spartan input**

![screenshot of spartan input functionality](https://hedy.smol.pub/spartan-input.png)

## remotes

- [sourcehut](https://sr.ht/~hedy/gelim)
- [tildegit (gitea)](https://tildegit.org/hedy/gelim)
- [github](https://github.com/hedyhli/gelim)
- [codeberg](https://codeberg.org/hedy/gelim)

## bugs, features, feedback, and contributions

**questions and general feedback**:

* send a ([plain text](https://useplaintext.email)) email to my
[public inbox](https://lists.sr.ht/~hedy/inbox).
* [How to subscribe to the mailing list without a sourcehut
  account](https://man.sr.ht/lists.sr.ht/#email-controls)
* or join `#gelim` on libera.chat irc for questions and suggestion

**bugs and feature requests**

* submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim).
* you don't need a sourcehut account to subscribe or submit a ticket, [here's
  how to do them with email](https://man.sr.ht/todo.sr.ht/#email-access)
* or you can also use the one on
  [github](https://github.com/hedyhli/gelim/issues) if you prefer.

**pull request, patches**

* send patches to my [public inbox](https://lists.sr.ht/~hedy/inbox)
* If you prefer pull requests instead,
  [this](https://github.com/hedyhli/gelim/pulls) is where PRs should go. You
  could also send PRs to my public inbox but I'll have to search up how to
  merge them (lol)

