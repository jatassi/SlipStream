// SlipStream Stub Launcher for Linux deb/rpm packages
//
// This minimal launcher enables auto-updates without requiring sudo.
// It bootstraps the real binary to a user-writable location on first run,
// then executes it. All subsequent updates happen in the user directory.
//
// Directory structure:
//   /usr/bin/slipstream                 - This stub (installed by package)
//   /usr/share/slipstream/slipstream    - Bundled binary (bootstrap source)
//   ~/.local/share/slipstream/
//     ├── bin/slipstream                - Real binary (auto-updated here)
//     ├── slipstream.db                 - Database
//     ├── config.yaml                   - Config
//     └── logs/                         - Logs

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

const (
	appName       = "slipstream"
	packageBinary = "/usr/share/slipstream/slipstream"
)

func main() {
	userDir, err := getUserDataDir()
	if err != nil {
		fatal("Failed to determine user data directory: %v", err)
	}

	userBinary := filepath.Join(userDir, "bin", appName)

	// Bootstrap if the user binary doesn't exist
	if !fileExists(userBinary) {
		if err := bootstrap(userBinary); err != nil {
			fatal("Failed to bootstrap: %v", err)
		}
	}

	// Execute the real binary, replacing this process
	// Pass through all arguments and environment
	if err := syscall.Exec(userBinary, os.Args, os.Environ()); err != nil {
		fatal("Failed to execute %s: %v", userBinary, err)
	}
}

func getUserDataDir() (string, error) {
	// Check XDG_DATA_HOME first, fall back to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, appName), nil
}

func bootstrap(userBinary string) error {
	fmt.Println("SlipStream: First run, setting up user directory...")

	// Create the bin directory
	binDir := filepath.Dir(userBinary)
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("create bin directory: %w", err)
	}

	// Copy the bundled binary to user directory
	if !fileExists(packageBinary) {
		return fmt.Errorf("bundled binary not found at %s", packageBinary)
	}

	if err := copyFile(packageBinary, userBinary); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}

	// Make it executable
	if err := os.Chmod(userBinary, 0o600); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	fmt.Println("SlipStream: Setup complete!")
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "SlipStream: "+format+"\n", args...)
	os.Exit(1)
}
