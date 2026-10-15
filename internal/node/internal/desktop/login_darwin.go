package desktop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/install"
)

func launchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", install.BundleID+".plist"), nil
}

// StartAtLogin reports the LaunchAgent, which the app writes on its first
// launch and the menu bar item flips.
func StartAtLogin() (bool, error) {
	p, err := launchAgentPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read the LaunchAgent: %w", err)
	}
	return true, nil
}

// SetStartAtLogin writes the LaunchAgent, pointing at the trampoline, or
// removes it. launchd reads it at the next login, so writing it never
// starts a second app now.
func SetStartAtLogin(on bool) error {
	p, err := launchAgentPath()
	if err != nil {
		return err
	}
	if !on {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove the LaunchAgent: %w", err)
		}
		return nil
	}
	program, err := trampoline()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { //nolint:gosec // G301: ~/Library/LaunchAgents is not secret
		return fmt.Errorf("write the LaunchAgent: %w", err)
	}
	if err := os.WriteFile(p, launchAgent(program), 0o644); err != nil { //nolint:gosec // G306: launchd reads the LaunchAgent
		return fmt.Errorf("write the LaunchAgent: %w", err)
	}
	return nil
}
