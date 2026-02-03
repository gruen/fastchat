VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build install uninstall clean test

build:
	go build -ldflags "-X main.version=$(VERSION)" -o ai-tui .

install: build
	./ai-tui install

uninstall: build
	./ai-tui uninstall

test:
	go test ./...

clean:
	rm -f ai-tui
