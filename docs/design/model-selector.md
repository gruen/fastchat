# Design: Model Selector TUI

## Problem

There's no way to switch models without editing `config.toml` and restarting.
Users with multiple providers/models configured need a fast way to pick one mid-session.

## Solution

A fuzzy-filter overlay (Ctrl+M) that lists all configured provider/model pairs and a persistent status bar showing the active model.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Trigger | Keybinding only (Ctrl+M) | Keeps startup fast; no mandatory prompt |
| Display format | `provider > model` | Shows both without clutter |
| List style | Fuzzy filter (bubbles/list) | Fast selection when many models configured |
| On switch | Start new session | Avoids mixed-model conversations in DB |
| Model source | Config only | Simple, predictable, no API calls needed |
| Status bar | Always visible | User always knows which model is active |

## Component Sketch

```
┌──────────────────────────────────────┐
│  assistant > claude-sonnet-4         │  <- status bar
├──────────────────────────────────────┤
│                                      │
│  (conversation area)                 │
│                                      │
├──────────────────────────────────────┤
│  > _                                 │  <- input area
└──────────────────────────────────────┘

Ctrl+M opens:

┌──────────────────────────────────────┐
│  Select model: son_                  │  <- filter input
├──────────────────────────────────────┤
│  > claude > claude-sonnet-4          │
│    claude > claude-haiku-3.5         │
│    openai > gpt-4o                   │
│    local  > llama3                   │
└──────────────────────────────────────┘
```

## Scope

**In scope:** selector overlay, status bar, new-session-on-switch logic.

**Out of scope:** freeform model input, API model discovery, continue-session-on-switch option.
