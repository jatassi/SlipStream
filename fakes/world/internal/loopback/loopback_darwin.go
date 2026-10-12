package loopback

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	alreadySetUp   = "loopback aliases already set up"
	alreadyRemoved = "loopback aliases already removed"
)

var errNotRoot = errors.New("setup-loopback needs root; run it with sudo")

// Setup writes the LaunchDaemon, bootstraps it and adds every alias now.
// It changes nothing when the LaunchDaemon is current and loaded and every
// alias exists.
func Setup(ctx context.Context) (string, error) {
	if os.Geteuid() != 0 {
		return "", errNotRoot
	}
	current, _ := os.ReadFile(PlistPath)
	loaded := isLoaded(ctx)
	present, err := presentAliases()
	if err != nil {
		return "", err
	}
	if bytes.Equal(current, Plist()) && loaded && len(present) == len(aliases()) {
		return alreadySetUp, nil
	}
	if loaded {
		if err := run(ctx, "/bin/launchctl", "bootout", "system/"+Label); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(PlistPath, Plist(), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", PlistPath, err)
	}
	if err := run(ctx, "/bin/launchctl", "bootstrap", "system", PlistPath); err != nil {
		return "", err
	}
	if err := run(ctx, "/bin/sh", "-c", program); err != nil {
		return "", err
	}
	present, err = presentAliases()
	if err != nil {
		return "", err
	}
	if len(present) != len(aliases()) {
		return "", fmt.Errorf("lo0 has only %v of the aliases after setup", present)
	}
	return fmt.Sprintf("set up loopback aliases 127.0.0.%d to 127.0.0.%d, kept at every boot by %s", first, last, PlistPath), nil
}

// Remove boots the LaunchDaemon out, deletes it and removes every alias.
func Remove(ctx context.Context) (string, error) {
	if os.Geteuid() != 0 {
		return "", errNotRoot
	}
	changed := false
	if isLoaded(ctx) {
		if err := run(ctx, "/bin/launchctl", "bootout", "system/"+Label); err != nil {
			return "", err
		}
		changed = true
	}
	switch err := os.Remove(PlistPath); {
	case err == nil:
		changed = true
	case !errors.Is(err, os.ErrNotExist):
		return "", fmt.Errorf("remove %s: %w", PlistPath, err)
	}
	present, err := presentAliases()
	if err != nil {
		return "", err
	}
	for _, a := range present {
		if err := run(ctx, "/sbin/ifconfig", "lo0", "-alias", a.String()); err != nil {
			return "", err
		}
		changed = true
	}
	if !changed {
		return alreadyRemoved, nil
	}
	return fmt.Sprintf("removed loopback aliases 127.0.0.%d to 127.0.0.%d and %s", first, last, PlistPath), nil
}

func isLoaded(ctx context.Context) bool {
	return exec.CommandContext(ctx, "/bin/launchctl", "print", "system/"+Label).Run() == nil
}

// presentAliases are the aliases lo0 has now.
func presentAliases() ([]netip.Addr, error) {
	lo, err := net.InterfaceByName("lo0")
	if err != nil {
		return nil, fmt.Errorf("lo0: %w", err)
	}
	addrs, err := lo.Addrs()
	if err != nil {
		return nil, fmt.Errorf("lo0: %w", err)
	}
	var have []netip.Addr
	for _, a := range addrs {
		if p, err := netip.ParsePrefix(a.String()); err == nil {
			have = append(have, p.Addr())
		}
	}
	var present []netip.Addr
	for _, a := range aliases() {
		if slices.Contains(have, a) {
			present = append(present, a)
		}
	}
	return present, nil
}

func run(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return nil
}

func aliases() []netip.Addr {
	out := make([]netip.Addr, 0, last-first+1)
	for n := first; n <= last; n++ {
		out = append(out, netip.AddrFrom4([4]byte{127, 0, 0, byte(n)}))
	}
	return out
}
