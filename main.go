package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mg/ai-tui/internal/config"
	"github.com/mg/ai-tui/internal/db"
	"github.com/mg/ai-tui/internal/install"
	"github.com/mg/ai-tui/internal/llm"
	"github.com/mg/ai-tui/internal/tui"
)

var version = "dev"

//go:embed config.example.toml
var defaultConfig []byte

//go:embed scripts/ai-tui-launch.sh
var launcherScript []byte

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("ai-tui %s\n", version)
			os.Exit(0)
		case "install":
			if err := runInstall(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "uninstall":
			if err := runUninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	runTUI()
}

func runInstall() error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}
	return install.Install(install.Options{
		Self:     self,
		Config:   defaultConfig,
		Launcher: launcherScript,
	})
}

func runUninstall() error {
	purge := len(os.Args) > 2 && os.Args[2] == "--purge"
	return install.Uninstall(install.Options{Purge: purge})
}

func runTUI() {
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
			fmt.Fprintf(os.Stderr, "Run 'ai-tui install' to set up config and launcher script.\n")
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
