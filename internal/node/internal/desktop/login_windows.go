package desktop

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"

	"github.com/jatassi/slipstream/internal/install"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// StartAtLogin reports the HKCU Run value, which the MSI authors and the
// tray flips.
func StartAtLogin() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf(`open HKCU\%s: %w`, runKey, err)
	}
	defer func() { _ = k.Close() }()
	_, _, err = k.GetStringValue(install.WindowsRunValue)
	switch {
	case errors.Is(err, registry.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf(`read HKCU\%s\%s: %w`, runKey, install.WindowsRunValue, err)
	}
	return true, nil
}

// SetStartAtLogin writes the Run value, pointing at the trampoline, or
// removes it.
func SetStartAtLogin(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf(`open HKCU\%s: %w`, runKey, err)
	}
	defer func() { _ = k.Close() }()
	if !on {
		if err := k.DeleteValue(install.WindowsRunValue); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf(`remove HKCU\%s\%s: %w`, runKey, install.WindowsRunValue, err)
		}
		return nil
	}
	program, err := trampoline()
	if err != nil {
		return err
	}
	if err := k.SetStringValue(install.WindowsRunValue, `"`+program+`"`); err != nil {
		return fmt.Errorf(`write HKCU\%s\%s: %w`, runKey, install.WindowsRunValue, err)
	}
	return nil
}
