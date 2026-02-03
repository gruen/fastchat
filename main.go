package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

func main() {
	var (
		configPath  = flag.String("config", "~/.config/ai-tui/config.toml", "Path to config file")
		showVersion = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("ai-tui %s\n", version)
		os.Exit(0)
	}

	// For now, just acknowledge the config path and exit
	_ = configPath
	fmt.Println("ai-tui starting...")
	os.Exit(0)
}
