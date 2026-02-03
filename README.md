# ai-tui

A terminal UI for chatting with LLMs, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea). Designed to run as a floating window on Hyprland (or any tiling WM) — hit a keybind, chat, dismiss.

## Features

- **Multi-provider support** — Claude (Anthropic API), OpenAI, and any OpenAI-compatible endpoint (Ollama, local models, etc.)
- **Streaming responses** — Real-time token streaming with SSE parsing for both Anthropic and OpenAI protocols
- **Conversation history** — SQLite-backed session storage with browsing, search, and archival
- **Markdown rendering** — Assistant responses rendered with [Glamour](https://github.com/charmbracelet/glamour)
- **Markdown export** — Save conversations to `~/ai-notes/` (configurable) as clean Markdown files
- **Configurable via TOML** — Environment variable expansion in config values (e.g. `$ANTHROPIC_API_KEY`)
- **Hyprland integration** — Launcher script and window rules for a floating overlay experience

## Providers

Configure one or more providers in `~/.config/ai-tui/config.toml`:

| Provider | Protocol | Example `base_url` |
|----------|----------|-------------------|
| Claude | Anthropic Messages API | `https://api.anthropic.com` |
| OpenAI | OpenAI Chat Completions | `https://api.openai.com/v1` |
| Ollama | OpenAI-compatible | `http://localhost:11434/v1` |

## Install

```bash
make install
```

This installs:
- `~/.local/bin/ai-tui` — the binary
- `~/.local/bin/ai-tui-launch.sh` — terminal launcher script
- `~/.config/ai-tui/config.toml` — config (copied from example if missing)

The SQLite database is created automatically at `~/.local/share/ai-tui/ai-tui.db`.

## Configuration

Copy and edit the example config:

```bash
cp config.example.toml ~/.config/ai-tui/config.toml
```

```toml
default_provider = "claude"

[providers.claude]
api_key = "$ANTHROPIC_API_KEY"
model = "claude-sonnet-4-20250514"
system_prompt = "You are a helpful assistant. Be concise."
max_tokens = 4096

[storage]
db_path = "~/.local/share/ai-tui/ai-tui.db"
notes_dir = "~/ai-notes/"
```

Environment variables in values (prefixed with `$`) are expanded at load time.

## Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `Enter` | Compose | Send message |
| `Esc` | Streaming | Cancel generation |
| `Ctrl+H` | Global | Toggle history view |
| `Ctrl+N` | Global | New conversation |
| `Ctrl+D` | Global | Quit |
| `s` | History | Export session to Markdown |
| `d` | History | Archive session |
| `a` | History | Toggle archived sessions |

## Hyprland Setup

Add to your Hyprland config:

```
windowrulev2 = float, class:^(ai-tui-float)$
windowrulev2 = size 60% 70%, class:^(ai-tui-float)$
windowrulev2 = center, class:^(ai-tui-float)$
windowrulev2 = dimaround, class:^(ai-tui-float)$
bind = $mainMod, SPACE, exec, ~/.local/bin/ai-tui-launch.sh
```

The launcher script supports `foot`, `kitty`, and `alacritty` via the `AI_TUI_TERMINAL` env var (defaults to `foot`).

## Development

```bash
make build    # Build binary
make test     # Run tests
make clean    # Remove binary
```

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework (Elm architecture)
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components (textarea, viewport, list)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure-Go SQLite (no CGo)
- [BurntSushi/toml](https://github.com/BurntSushi/toml) — TOML config parsing
