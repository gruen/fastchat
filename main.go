package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/llm"
	"github.com/mg/ai-tui/internal/tui"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "Path to config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ai-tui %s\n", version)
		os.Exit(0)
	}

	// Determine config path
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}
	if strings.HasPrefix(cfgPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
			os.Exit(1)
		}
		cfgPath = filepath.Join(home, cfgPath[2:])
	}

	// Load config
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Config not found at %s\n", cfgPath)
			fmt.Fprintf(os.Stderr, "Create one with: mkdir -p ~/.config/ai-tui && cp config.example.toml ~/.config/ai-tui/config.toml\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	// Open database
	database, err := db.Open(cfg.Storage.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Build LLM providers
	providers := llm.BuildProviders(cfg.Providers)
	if len(providers) == 0 {
		fmt.Fprintf(os.Stderr, "error: no providers configured\n")
		os.Exit(1)
	}
	if _, ok := providers[cfg.DefaultProvider]; !ok {
		fmt.Fprintf(os.Stderr, "error: default provider %q not found\n", cfg.DefaultProvider)
		os.Exit(1)
	}

	// Create and run TUI
	model := tui.NewAppModel(cfg, database, providers)
	p := tea.NewProgram(&model, tea.WithAltScreen())
	model.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
