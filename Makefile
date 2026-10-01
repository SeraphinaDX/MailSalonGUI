GO ?= go
ZGO ?=

.PHONY: build run demo test check clean
build:
	GO="$(GO)" ZGO="$(ZGO)" sh ./scripts/go-toolchain.sh build -buildvcs=false -trimpath -o MailSalonGUI ./cmd/MailSalonGUI
run: build
	./MailSalonGUI
demo: build
	./MailSalonGUI -demo
test:
	$(GO) test -tags ci ./...
check:
	$(GO) vet -tags ci ./...
clean:
	rm -f MailSalonGUI MailSalonGUI.exe
