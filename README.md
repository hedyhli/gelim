# gelim

a minimalist line-mode gemini client, written in go.

**WARNING: the project is still in its early stages, so do expect bugs and
incomplete features, if you encounter them, or would like to suggest an
improvement, feel free to submit to the [ticket tracker](https://todo.sr.ht/~hedy/gelim).**


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

## bugs, features

- questions, general feedback, send a ([plain text](https://useplaintext.email)) email to my
[public inbox](mailto:~hedy/inbox@lists.sr.ht).
- bug, feature requests, submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim)
