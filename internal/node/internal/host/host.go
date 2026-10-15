// Package host reads the facts about the host a data dir sits on: who owns
// it, whether it is on a network share, and whether the OS meets the floor.
package host

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Owner is the user who owns a data dir when that is not this process's.
type Owner struct {
	UID  int
	Name string
}

// OnNetworkShare reports whether dir, or its nearest existing ancestor
// while dir does not exist yet, is on a network filesystem. That ancestor
// must be a directory: Windows reports a path below a file as not existing,
// where Linux and macOS fail the stat.
func OnNetworkShare(dir string) (bool, error) {
	for {
		info, err := os.Stat(dir)
		if err == nil && !info.IsDir() {
			return false, fmt.Errorf("data dir %s: not a directory", dir)
		}
		if err == nil {
			return networkFS(dir)
		}
		parent := filepath.Dir(dir)
		if !errors.Is(err, fs.ErrNotExist) || parent == dir {
			return false, fmt.Errorf("data dir %s: %w", dir, err)
		}
		dir = parent
	}
}

// version is a dotted version's major and minor numbers.
type version struct{ major, minor int }

func (v version) String() string { return fmt.Sprintf("%d.%d", v.major, v.minor) }

// parseVersion reads the leading major.minor of s, such as a kernel
// release 4.18.0-553.el8.x86_64.
func parseVersion(s string) (version, error) {
	major, rest, _ := strings.Cut(s, ".")
	end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
	if end < 0 {
		end = len(rest)
	}
	maj, err1 := strconv.Atoi(major)
	minor, err2 := strconv.Atoi(rest[:end])
	if err1 != nil || err2 != nil {
		return version{}, fmt.Errorf("%q is not a version", s)
	}
	return version{maj, minor}, nil
}
