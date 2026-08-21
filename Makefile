PREFIX ?= /usr/local
DESTDIR ?=
VERSION ?= 0.1.1
BUILD_DIR ?= build
BINARY := $(BUILD_DIR)/clone

.PHONY: build format install uninstall test lint deb

build:
	mkdir -p "$(BUILD_DIR)"
	go build -trimpath -ldflags "-X main.version=$(VERSION)" -o "$(BINARY)" ./cmd/clone

install: build
	install -Dm755 "$(BINARY)" "$(DESTDIR)$(PREFIX)/bin/clone"
	install -Dm644 completions/clone.bash "$(DESTDIR)$(PREFIX)/share/bash-completion/completions/clone"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/clone"
	rm -f "$(DESTDIR)$(PREFIX)/share/bash-completion/completions/clone"

test:
	go test ./...

format:
	gofmt -w cmd/clone/main.go cmd/clone/main_test.go

lint:
	test -z "$$(gofmt -d cmd/clone/main.go cmd/clone/main_test.go)"
	go vet ./...

deb: build
	scripts/build-deb "$(VERSION)"
