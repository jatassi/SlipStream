package host

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/netmodel"
)

const socketfilterfw = "/usr/libexec/ApplicationFirewall/socketfilterfw"

// alfGlobal is the application firewall's global state.
type alfGlobal struct {
	Off, BlockAll bool
	// AllowBuiltIn and AllowDownloaded are whether signed software is
	// allowed without an entry.
	AllowBuiltIn, AllowDownloaded bool
}

// alfApp is what the application firewall knows of one program.
type alfApp struct {
	Path            string
	Listed, Blocked bool
	// BuiltIn is signed by Apple, Downloaded by a Developer ID.
	BuiltIn, Downloaded bool
}

// Firewall reads the macOS application firewall through socketfilterfw's
// getters, which need no privilege. pf needs root to read and isn't
// read. When socketfilterfw can't be read every answer stays Unknown.
func Firewall(ctx context.Context, _ string, asks []netmodel.Rule) ([]netmodel.Rule, error) {
	socks, err := tcpListeners(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]netmodel.Rule, len(asks))
	for i, a := range asks {
		out[i] = a
		out[i].Tool = netmodel.SocketFilter
	}
	global, err := readALFGlobal(ctx)
	if err != nil {
		return out, nil //nolint:nilerr // an unread firewall reads Unknown
	}
	apps := map[string]alfApp{}
	for i, a := range out {
		path := ownerOf(socks, a.Port, a.Family)
		if _, ok := apps[path]; !ok && path != "" && !global.Off && !global.BlockAll {
			app, err := readALFApp(ctx, path)
			if err != nil {
				continue
			}
			apps[path] = app
		}
		out[i].Policy, out[i].Because = alfDecide(global, apps[path], a.Port)
	}
	return out, nil
}

// alfDecide is the application firewall's answer for port, owned by app.
// An unsigned program with no entry waits on a prompt, which blocks it.
func alfDecide(g alfGlobal, app alfApp, port uint16) (policy netmodel.Policy, because string) {
	switch {
	case g.Off:
		return netmodel.Allows, "The macOS firewall is off."
	case g.BlockAll:
		return netmodel.Blocks, "The macOS firewall blocks all incoming connections."
	case app.Path == "":
		return netmodel.PolicyUnknown, fmt.Sprintf("SlipStream can't see which program listens on port %d.", port)
	case app.Listed && app.Blocked:
		return netmodel.Blocks, fmt.Sprintf("The macOS firewall blocks incoming connections to %s.", app.Path)
	case app.Listed:
		return netmodel.Allows, fmt.Sprintf("The macOS firewall allows incoming connections to %s.", app.Path)
	case app.BuiltIn && g.AllowBuiltIn, app.Downloaded && g.AllowDownloaded:
		return netmodel.Allows, fmt.Sprintf("The macOS firewall allows signed software automatically, and %s is signed.", app.Path)
	}
	return netmodel.Blocks, app.Path + " isn't in the macOS firewall's list of apps."
}

func readALFGlobal(ctx context.Context) (alfGlobal, error) {
	state, err1 := socketFilter(ctx, "--getglobalstate")
	blockAll, err2 := socketFilter(ctx, "--getblockall")
	signed, err3 := socketFilter(ctx, "--getallowsigned")
	for _, err := range []error{err1, err2, err3} {
		if err != nil {
			return alfGlobal{}, err
		}
	}
	return parseALFGlobal(state, blockAll, signed), nil
}

// parseALFGlobal reads --getglobalstate ("Firewall is enabled. (State =
// 1)", 2 blocking all), --getblockall and --getallowsigned, whose lines
// end in ENABLED or DISABLED.
func parseALFGlobal(state, blockAll, signed string) alfGlobal {
	var g alfGlobal
	_, n, _ := strings.Cut(state, "State = ")
	level, _ := strconv.Atoi(strings.TrimRight(strings.TrimSpace(n), ")"))
	g.Off, g.BlockAll = level == 0, level == 2 || enabled(blockAll)
	for line := range strings.SplitSeq(signed, "\n") {
		l := strings.ToLower(line)
		switch {
		case strings.Contains(l, "built-in"):
			g.AllowBuiltIn = enabled(line)
		case strings.Contains(l, "downloaded"):
			g.AllowDownloaded = enabled(line)
		}
	}
	return g
}

func enabled(s string) bool {
	l := strings.ToLower(s)
	return strings.Contains(l, "enabled") && !strings.Contains(l, "disabled")
}

func readALFApp(ctx context.Context, path string) (alfApp, error) {
	out, err := socketFilter(ctx, "--getappblocked", path)
	if err != nil {
		return alfApp{}, err
	}
	app := parseALFApp(path, out)
	if !app.Listed {
		app.BuiltIn = signedBy(ctx, "anchor apple", path)
		app.Downloaded = !app.BuiltIn && signedBy(ctx, "anchor apple generic", path)
	}
	return app, nil
}

// parseALFApp reads --getappblocked: a listed program is permitted or
// blocked, and any other answer is a program with no entry.
func parseALFApp(path, out string) alfApp {
	l := strings.ToLower(out)
	app := alfApp{Path: path}
	switch {
	case strings.Contains(l, "block"):
		app.Listed, app.Blocked = true, true
	case strings.Contains(l, "permit"), strings.Contains(l, "allow"):
		app.Listed = true
	}
	return app
}

func socketFilter(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, socketfilterfw, args...).Output() //nolint:gosec // G204: socketfilterfw's getters, one with the listener's path
	if err != nil {
		return "", fmt.Errorf("run socketfilterfw %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// signedBy is whether codesign verifies path against requirement.
func signedBy(ctx context.Context, requirement, path string) bool {
	return exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "-R="+requirement, path).Run() == nil //nolint:gosec // G204: the listener's executable
}
