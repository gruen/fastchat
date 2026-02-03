#!/usr/bin/env bash
# ai-tui-launch.sh - Launch ai-tui in a floating terminal for Hyprland
# Set AI_TUI_TERMINAL to your preferred terminal (foot, kitty, alacritty)

TERMINAL="${AI_TUI_TERMINAL:-foot}"
CLASS="ai-tui-float"

case "$TERMINAL" in
    foot)
        exec foot --app-id="$CLASS" -e ai-tui ;;
    kitty)
        exec kitty --class="$CLASS" -e ai-tui ;;
    alacritty)
        exec alacritty --class "$CLASS","$CLASS" -e ai-tui ;;
    *)
        echo "Unsupported terminal: $TERMINAL" >&2
        echo "Set AI_TUI_TERMINAL to one of: foot, kitty, alacritty" >&2
        exit 1
        ;;
esac
