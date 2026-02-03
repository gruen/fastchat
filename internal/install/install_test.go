package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstall(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	configDir := filepath.Join(tmp, "config")

	// Create a fake "self" binary to copy
	selfPath := filepath.Join(tmp, "ai-tui-src")
	if err := os.WriteFile(selfPath, []byte("fake-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		BinDir:    binDir,
		ConfigDir: configDir,
		Config:    []byte("# test config\n"),
		Launcher:  []byte("#!/bin/bash\necho test\n"),
		Self:      selfPath,
	}

	if err := Install(opts); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	// Verify binary was copied
	binPath := filepath.Join(binDir, "ai-tui")
	if _, err := os.Stat(binPath); err != nil {
		t.Errorf("binary not found at %s", binPath)
	}

	// Verify launcher was written
	launcherPath := filepath.Join(binDir, "ai-tui-launch.sh")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("launcher not found: %v", err)
	}
	if string(data) != "#!/bin/bash\necho test\n" {
		t.Errorf("launcher content mismatch: %q", data)
	}
	info, _ := os.Stat(launcherPath)
	if info.Mode().Perm()&0111 == 0 {
		t.Error("launcher is not executable")
	}

	// Verify config was written
	configPath := filepath.Join(configDir, "config.toml")
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config not found: %v", err)
	}
	if string(data) != "# test config\n" {
		t.Errorf("config content mismatch: %q", data)
	}
}

func TestInstallSkipsExistingConfig(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	configDir := filepath.Join(tmp, "config")

	selfPath := filepath.Join(tmp, "ai-tui-src")
	if err := os.WriteFile(selfPath, []byte("fake-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-create config
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("# existing config\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		BinDir:    binDir,
		ConfigDir: configDir,
		Config:    []byte("# new config\n"),
		Launcher:  []byte("#!/bin/bash\n"),
		Self:      selfPath,
	}

	if err := Install(opts); err != nil {
		t.Fatalf("Install() error: %v", err)
	}

	// Config should NOT be overwritten
	data, _ := os.ReadFile(configPath)
	if string(data) != "# existing config\n" {
		t.Errorf("existing config was overwritten: %q", data)
	}
}

func TestInstallSamePath(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Self is already at the install location
	selfPath := filepath.Join(binDir, "ai-tui")
	if err := os.WriteFile(selfPath, []byte("fake-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		BinDir:    binDir,
		ConfigDir: filepath.Join(tmp, "config"),
		Config:    []byte("# config\n"),
		Launcher:  []byte("#!/bin/bash\n"),
		Self:      selfPath,
	}

	if err := Install(opts); err != nil {
		t.Fatalf("Install() error: %v", err)
	}
}

func TestUninstallPurge(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	configDir := filepath.Join(tmp, "config")
	dataDir := filepath.Join(tmp, "data")

	// Create files to uninstall
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(configDir, 0755)
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(binDir, "ai-tui"), []byte("bin"), 0755)
	os.WriteFile(filepath.Join(binDir, "ai-tui-launch.sh"), []byte("sh"), 0755)
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("cfg"), 0644)
	os.WriteFile(filepath.Join(dataDir, "ai-tui.db"), []byte("db"), 0644)

	opts := Options{
		BinDir:    binDir,
		ConfigDir: configDir,
		DataDir:   dataDir,
		Purge:     true,
	}

	if err := Uninstall(opts); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}

	// All should be gone
	if _, err := os.Stat(filepath.Join(binDir, "ai-tui")); !os.IsNotExist(err) {
		t.Error("binary was not removed")
	}
	if _, err := os.Stat(filepath.Join(binDir, "ai-tui-launch.sh")); !os.IsNotExist(err) {
		t.Error("launcher was not removed")
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Error("config dir was not removed")
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Error("data dir was not removed")
	}
}

func TestUninstallNothingToRemove(t *testing.T) {
	tmp := t.TempDir()
	opts := Options{
		BinDir:    filepath.Join(tmp, "bin"),
		ConfigDir: filepath.Join(tmp, "config"),
		DataDir:   filepath.Join(tmp, "data"),
		Purge:     true,
	}

	// Should not error on missing files
	if err := Uninstall(opts); err != nil {
		t.Fatalf("Uninstall() error: %v", err)
	}
}
