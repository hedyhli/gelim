# gelim

a minimalist line-mode gemini client, written in go.

**WARNING: the project is still in its early stages, so do expect bugs and
incomplete features, if you encounter them, or would like to suggest an
improvement, feel free to submit to the [ticket tracker](https://todo.sr.ht/~hedy/gelim).**

![gelim demo image](https://hedy.ftp.sh/gelim-pager.png)
![another one](https://hedy.ftp.sh/gelim-cmds.png)

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

## bugs, features, feedback

- **questions, general feedback**: send a ([plain text](https://useplaintext.email)) email to my
[public inbox](https://lists.sr.ht/~hedy/inbox). [How to subscribe to the mailing list without a sourcehut account](https://man.sr.ht/lists.sr.ht/#email-controls)
- **bugs, feature requests**: submit a ticket to the [tracker](https://todo.sr.ht/~hedy/gelim).
you don't need a sourcehut account to subscribe or submit a ticket, [here's how to do them with email](https://man.sr.ht/todo.sr.ht/#email-access)
