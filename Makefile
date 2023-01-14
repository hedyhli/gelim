# Referenced from aerc's makefile

.POSIX:
.SUFFIXES:
.SUFFIXES: .1 .1.scd

TAG      != git describe --abbrev=0 --tags
REVISION != git rev-parse --short HEAD
VERSION   = $(TAG)-$(REVISION)

PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
MANDIR?=$(PREFIX)/share/man
GO?=go
GOFLAGS?=
LDFLAGS:=-X main.Prefix=$(PREFIX)
LDFLAGS+=-X main.Version=$(VERSION)

GOSRC:=$(shell find * -name '*.go')
GOSRC+=go.mod go.sum

DOCS := gelim.1


.PHONY: all
all: gelim $(DOCS)

gelim: $(GOSRC)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: checkfmt
checkfmt:
	@if [ `gofmt -l . | wc -l` -ne 0 ]; then \
		gofmt -d .; \
		echo "ERROR: source files need reformatting with gofmt"; \
		exit 1; \
	fi

.1.scd.1:
	scdoc < $< > $@

.PHONY: doc
doc: $(DOCS)

# Exists in GNUMake but not in NetBSD make and others.
RM?=rm -f

.PHONY: clean
clean:
	$(RM) $(DOCS) gelim

.PHONY: install
install: $(DOCS) gelim
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(MANDIR)/man1
	install -m755 gelim $(DESTDIR)$(BINDIR)/gelim
	install -m644 gelim.1 $(DESTDIR)$(MANDIR)/man1/gelim.1

.PHONY: checkinstall
checkinstall:
	$(DESTDIR)$(BINDIR)/gelim --version
	test -e $(DESTDIR)$(MANDIR)/man1/gelim.1
	@echo OK

RMDIR_IF_EMPTY:=sh -c '! [ -d $$0 ] || ls -1qA $$0 | grep -q . || rmdir $$0'

.PHONY: uninstall
uninstall:
	$(RM) $(DESTDIR)$(BINDIR)/gelim
	$(RM) $(DESTDIR)$(MANDIR)/man1/gelim.1
	$(RM) -r $(DESTDIR)$(SHAREDIR)
	${RMDIR_IF_EMPTY} $(DESTDIR)$(BINDIR)
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)/man1
	$(RMDIR_IF_EMPTY) $(DESTDIR)$(MANDIR)
