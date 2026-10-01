GO ?= go
ZGO ?=
PREFIX ?= $(HOME)/.local
DATADIR ?= $(PREFIX)/share
DESTDIR ?=
export PREFIX DATADIR DESTDIR

.PHONY: build run demo install uninstall test check clean
build:
	GO="$(GO)" ZGO="$(ZGO)" sh ./scripts/go-toolchain.sh build -buildvcs=false -trimpath -o MailSalonGUI ./cmd/MailSalonGUI
run: build
	./MailSalonGUI
demo: build
	./MailSalonGUI -demo
install: build
	sh ./scripts/install-desktop.sh install
uninstall:
	sh ./scripts/install-desktop.sh uninstall
test:
	$(GO) test -tags ci ./...
check:
	$(GO) vet -tags ci ./...
clean:
	rm -f MailSalonGUI MailSalonGUI.exe
