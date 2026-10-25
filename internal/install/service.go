package install

import (
	"fmt"
	"strconv"
)

// Step is one action of a user service plan. A plan is pure, so it is
// tested as data and applied afterwards in order.
type Step interface{ step() }

type WriteFile struct{ Path, Content string }

type RemoveFile struct{ Path string }

type Command struct{ Args []string }

// AwaitRunning waits until the Node answers status on the plan's data dir.
type AwaitRunning struct{}

func (WriteFile) step()    {}
func (RemoveFile) step()   {}
func (Command) step()      {}
func (AwaitRunning) step() {}

func systemctl(args ...string) Command {
	return Command{Args: append([]string{"systemctl", "--user"}, args...)}
}

// Linger is whether systemd keeps a user's services running while the
// user has no login session.
type Linger struct {
	UID int
	On  bool
}

// UserServiceInstall plans service install --user: the unit runs k's
// binary on dataDir, an absolute path. current is the unit file there now,
// nil when there is none. A rerun with an identical unit rewrites and
// restarts nothing, and still starts a unit someone stopped.
func UserServiceInstall(k Kind, l Layout, dataDir string, current []byte, linger Linger) ([]Step, error) {
	p, err := UserUnitPath(k, l)
	if err != nil {
		return nil, err
	}
	exe := l.Executable
	if k == AppImage {
		// An AppImage runs from a mount under /tmp that dies with it.
		if exe = l.Getenv("APPIMAGE"); !l.isAbs(exe) {
			return nil, fmt.Errorf("APPIMAGE is %q, not an absolute path. Start service install --user from the AppImage", exe)
		}
	}
	unit := UserUnit(exe, dataDir)
	steps := []Step{systemctl("daemon-reload"), systemctl("enable", "--now", UnitName)}
	if current == nil || string(current) != unit {
		steps = []Step{WriteFile{Path: p, Content: unit}, systemctl("daemon-reload"), systemctl("enable", UnitName), systemctl("restart", UnitName)}
	}
	// logind asks polkit even when linger is already on, and polkit refuses
	// a caller with no login session, such as cron or a CI runner. Without
	// a user, loginctl names the caller's session's user, so it names the
	// UID.
	if !linger.On {
		steps = append(steps, Command{Args: []string{"loginctl", "enable-linger", strconv.Itoa(linger.UID)}})
	}
	return append(steps, AwaitRunning{}), nil
}

// UserServiceUninstall plans service uninstall --user. exists reports
// whether the unit file is there. It leaves linger on, since other user
// services may need it.
func UserServiceUninstall(k Kind, l Layout, exists bool) ([]Step, error) {
	p, err := UserUnitPath(k, l)
	if err != nil || !exists {
		return nil, err
	}
	return []Step{systemctl("disable", "--now", UnitName), RemoveFile{Path: p}, systemctl("daemon-reload")}, nil
}

// UserUnitPath is where service install --user writes k's unit on l.
func UserUnitPath(k Kind, l Layout) (string, error) {
	if !k.SelfInstalled() {
		return "", fmt.Errorf("a user service is for a tarball or AppImage install, and this is a %s install", k)
	}
	if dir := l.Getenv("XDG_CONFIG_HOME"); l.isAbs(dir) {
		return l.join(dir, "systemd", "user", UnitName), nil
	}
	return l.under("HOME", ".config", "systemd", "user", UnitName)
}
