GO ?= go
ZGO ?=
PREFIX ?= $(HOME)/.local
DATADIR ?= $(PREFIX)/share
DESTDIR ?=
export PREFIX DATADIR DESTDIR

CORE_PACKAGES = ./internal/config ./internal/maildir ./internal/mimeutil ./internal/transport ./internal/pgp ./internal/pim ./internal/drafts ./internal/version

.PHONY: build build-gui build-tui build-all run demo install uninstall test test-tui check clean
build: build-gui
build-gui:
	GO="$(GO)" ZGO="$(ZGO)" sh ./scripts/go-toolchain.sh build -buildvcs=false -trimpath -o MailSalonGUI ./cmd/MailSalonGUI
build-tui:
	CGO_ENABLED=0 $(GO) build -buildvcs=false -trimpath -o MailSalon ./cmd/MailSalon
build-all: build-gui build-tui
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
test-tui:
	CGO_ENABLED=0 $(GO) test $(CORE_PACKAGES) ./internal/ui ./cmd/MailSalon
check:
	$(GO) vet -tags ci ./...
clean:
	rm -f MailSalonGUI MailSalonGUI.exe MailSalon MailSalon.exe
