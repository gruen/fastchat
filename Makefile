VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build install clean test

build:
	go build -ldflags "-X main.version=$(VERSION)" -o ai-tui .

install: build
	install -Dm755 ai-tui $(HOME)/.local/bin/ai-tui
	install -Dm755 scripts/ai-tui-launch.sh $(HOME)/.local/bin/ai-tui-launch.sh
	@mkdir -p $(HOME)/.config/ai-tui
	@test -f $(HOME)/.config/ai-tui/config.toml || \
		install -Dm644 config.example.toml $(HOME)/.config/ai-tui/config.toml

test:
	go test ./...

clean:
	rm -f ai-tui
