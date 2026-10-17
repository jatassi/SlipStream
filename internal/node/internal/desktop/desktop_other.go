//go:build !windows && !darwin

package desktop

import "errors"

// HasLoginItem reports whether the OS has a login item the app manages.
const HasLoginItem = false

// RestartOnMainThread is false: with no tray, any goroutine may exec.
const RestartOnMainThread = false

// Run returns once app is done. Linux shows no tray, as in v1, so app
// differs from run only by the browser hand-off.
func Run(app App, opts Options) {
	opts.Log.Info("tray", "state", TrayNone)
	<-app.Done()
}

func Tray() TrayState { return TrayNone }

// Alert shows nothing: Linux reports a failed start on stderr alone.
func Alert(string, string) {}

func StartAtLogin() (bool, error) { return false, errors.ErrUnsupported }

func SetStartAtLogin(bool) error { return errors.ErrUnsupported }
