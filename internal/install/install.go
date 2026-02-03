package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Options configures install/uninstall paths. Zero values use defaults.
type Options struct {
	BinDir    string // default: ~/.local/bin
	ConfigDir string // default: ~/.config/ai-tui
	DataDir   string // default: ~/.local/share/ai-tui
	Config    []byte // embedded config.example.toml
	Launcher  []byte // embedded ai-tui-launch.sh
	Self      string // path to current executable
	Purge     bool   // for uninstall: remove config and data without prompting
}

func (o *Options) binDir() string {
	if o.BinDir != "" {
		return o.BinDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

func (o *Options) configDir() string {
	if o.ConfigDir != "" {
		return o.ConfigDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ai-tui")
}

func (o *Options) dataDir() string {
	if o.DataDir != "" {
		return o.DataDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "ai-tui")
}

// Install copies the binary, launcher script, and default config to their expected locations.
func Install(opts Options) error {
	binDir := opts.binDir()
	configDir := opts.configDir()
	binPath := filepath.Join(binDir, "ai-tui")
	launcherPath := filepath.Join(binDir, "ai-tui-launch.sh")
	configPath := filepath.Join(configDir, "config.toml")

	// Ensure bin directory exists
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create bin directory: %w", err)
	}

	// Copy binary (skip if same path)
	selfResolved, err := filepath.Abs(opts.Self)
	if err != nil {
		return fmt.Errorf("resolve self path: %w", err)
	}
	binResolved, _ := filepath.Abs(binPath)
	if selfResolved == binResolved {
		fmt.Printf("  ✓ Binary already at %s\n", shortPath(binPath))
	} else {
		if err := copyFile(opts.Self, binPath, 0755); err != nil {
			return fmt.Errorf("install binary: %w", err)
		}
		fmt.Printf("  ✓ Installed binary to %s\n", shortPath(binPath))
	}

	// Write launcher script
	if err := os.WriteFile(launcherPath, opts.Launcher, 0755); err != nil {
		return fmt.Errorf("install launcher: %w", err)
	}
	fmt.Printf("  ✓ Installed launcher to %s\n", shortPath(launcherPath))

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Write default config (only if not exists)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, opts.Config, 0644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("  ✓ Wrote default config to %s\n", shortPath(configPath))
		fmt.Println("    (edit this file to add your API keys)")
	} else {
		fmt.Printf("  - Config already exists at %s (skipped)\n", shortPath(configPath))
	}

	fmt.Println()
	fmt.Println("Add to your Hyprland config:")
	fmt.Println()
	fmt.Println("  windowrulev2 = float, class:^(ai-tui-float)$")
	fmt.Println("  windowrulev2 = size 60% 70%, class:^(ai-tui-float)$")
	fmt.Println("  windowrulev2 = center, class:^(ai-tui-float)$")
	fmt.Println("  windowrulev2 = dimaround, class:^(ai-tui-float)$")
	fmt.Printf("  bind = $mainMod, SPACE, exec, %s\n", shortPath(launcherPath))

	return nil
}

// Uninstall removes installed files. With Purge, also removes config and data.
func Uninstall(opts Options) error {
	binDir := opts.binDir()
	configDir := opts.configDir()
	dataDir := opts.dataDir()
	binPath := filepath.Join(binDir, "ai-tui")
	launcherPath := filepath.Join(binDir, "ai-tui-launch.sh")

	removed := false

	if removeIfExists(binPath) {
		fmt.Printf("  ✓ Removed %s\n", shortPath(binPath))
		removed = true
	}
	if removeIfExists(launcherPath) {
		fmt.Printf("  ✓ Removed %s\n", shortPath(launcherPath))
		removed = true
	}

	removeData := opts.Purge
	if !removeData {
		configExists := dirExists(configDir)
		dataExists := dirExists(dataDir)
		if configExists || dataExists {
			fmt.Print("  Remove config and data? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			removeData = strings.TrimSpace(strings.ToLower(answer)) == "y"
		}
	}

	if removeData {
		if removeDirIfExists(configDir) {
			fmt.Printf("  ✓ Removed %s\n", shortPath(configDir))
			removed = true
		}
		if removeDirIfExists(dataDir) {
			fmt.Printf("  ✓ Removed %s\n", shortPath(dataDir))
			removed = true
		}
	}

	if !removed {
		fmt.Println("  Nothing to remove.")
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func removeIfExists(path string) bool {
	if err := os.Remove(path); err == nil {
		return true
	}
	return false
}

func removeDirIfExists(path string) bool {
	if err := os.RemoveAll(path); err == nil {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func shortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
