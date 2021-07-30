# gelim

a minimalist line-mode gemini client, written in go.

**WARNING: the project is still in its early stages, so do expect bugs and
incomplete features, if you encounter them, or would like to suggest an
improvement, feel free to submit to the [ticket tracker](https://todo.sr.ht/~hedy/gelim).**


<table><tr>
<td> <img src="https://hedy.smol.pub/gelim-cmds.png" alt="screenshot" style="width: 250px;"/> </td>
<td> <img src="https://hedy.smol.pub/gelim-pager.png" alt="screenshot" style="width: 250px;"/> </td>
</tr></table>


## features

- searching from the command line
- inputs from the command line
- relative url at prompt
- pager (requires less(1))
- configuration file
  - custom search URL
  - custom pager opts
  - and more!
- [spartan:// protocol](gemini://spartan.mozz.us) support
- check out some of the planned features in the [ticket tracker](https://todo.sr.ht/~hedy/gelim)

## install

I plan to set up automated uploads of pre-built binaries to each
[release](https://git.sr.ht/~hedy/gelim/refs) at some point in the future. at the moment
you can clone the repo and `go build`:

```
git clone https://git.sr.ht/~hedy/gelim
cd gelim
# git checkout v0.0.0  # pin specific version or commit
go build
```

and move the `gelim` binary somewhere in your $PATH (like `/usr/local/bin`)

I could also write a Makefile, and have the build put $VERSION number in there or something
too, [let me know](mailto:~hedy/inbox@lists.sr.ht) if you'd like that since I'm not wanting
to do that yet.

## usage

use [scdoc](https://sr.ht/~sircmpwn/scdoc) to compile [gelim.1.scd](gelim.1.scd) and put it in
your man path

I'm also planning to have a mirror of that manual hosted on man.sr.ht

Note that the manpage may not be the most recently updated. But new features and things like that
will definetely be put in there once it's tested stable.

### quickstart

```
gelim gemini.circumlunar.space
```
This will bring you to less(1). You can use less to browse the page normally.

*Note: if you see something like "mouse is not an option", don't panic, this is
because your system does not support one of gelim's default less options, you
should skip over to the 'config' section below, and configure your lessOpts to
remove the mouse option, and any other your version of less doesn't have.*

When you want to visit a link, you have to quit less first.
```
q
```
The page will be fetched and you'll be in less again.

Now let's try something more interesting.

While you're at the prompt type:
```
rawtext.club
```
Say you don't have an account on RTC yet and would like to sign up.

Go to the bottom of the page, where the link to signing up is provided:
```
G
```
Then, you have to quit the pager:
```
q
```
Look for the link number that links to the sign up page, and enter it directy at the prompt.
As of writing, the link number is 38, but keep in mind this number may change when you are
trying this out.
```
38
```
And now you've decided to have a look at rawtext.club's values at the front page on more time.
Unfortunately, the sign up page does not provide a link to go back to home. No worries, you can
directly use the path (prefixed with . or /) at the prompt.

Let's try it out. Quit the pager (`q`), and enter:
```
/
```
Voila, you're at the front page again!

Thanks for trying out this quickstart tutorial, there is still much to explore. Type in `help`
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
lessOpts = "-FSXR~"       # default: "-FSXR~ --mouse -P pager (q to quit)"

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

## remotes

- [sourcehut](https://sr.ht/~hedy/gelim)
- [tildegit (gitea)](https://tildegit.org/hedy/gelim)

## bugs, features, feedback, contribution

- **questions, general feedback**: send a ([plain text](https://useplaintext.email)) email to my
[public inbox](https://lists.sr.ht/~hedy/inbox). [How to subscribe to the mailing list without a sourcehut account](https://man.sr.ht/lists.sr.ht/#email-controls)
- **bugs, feature requests**: submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim).
you don't need a sourcehut account to subscribe or submit a ticket, [here's how to do them with email](https://man.sr.ht/todo.sr.ht/#email-access)
- **pull request, patches**: send patches to my [public inbox](https://lists.sr.ht/~hedy/inbox). If you prefer pull requests instead, [this](https://tildegit.org/hedy/gelim/pulls) is where PRs should go. You could also send PRs to my public inbox but I'll have to search up how to merge them (lol)

