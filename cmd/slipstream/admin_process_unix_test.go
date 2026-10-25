//go:build unix

package main

import (
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/node"
)

func TestAdminResetRefusesADataDirAnotherUserOwns(t *testing.T) {
	t.Parallel()
	if os.Getuid() != 0 {
		t.Skip("only root can give a data dir to another user")
	}
	dir := dataDir(t)
	if err := os.Chown(dir, 65534, 65534); err != nil {
		t.Fatal(err)
	}
	owner, err := user.LookupId("65534")
	if err != nil {
		t.Fatal(err)
	}
	res := reset(t, dir, "admin@example.com")
	want := "slipstream: admin reset: data dir " + dir + " belongs to " + owner.Username + ". Run: sudo -u " + owner.Username + " slipstream admin reset ...\n"
	if res.code != node.ExitUsage || res.stdout != "" || !strings.HasPrefix(res.stderr, want) {
		t.Fatalf("foreign data dir: exit %d, stdout %q, stderr %q; want exit %d starting %q", res.code, res.stdout, res.stderr, node.ExitUsage, want)
	}
}
