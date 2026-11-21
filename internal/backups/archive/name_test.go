package archive_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/backups/archive"
)

func TestNamesRoundTripAndRefusePaths(t *testing.T) {
	m := manifest()
	m.BackupID[13], m.BackupID[14], m.BackupID[15] = 0x03, 0xcc, 0x74
	name := archive.NameOf(m)
	if want := "slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age"; name.String() != want {
		t.Fatalf("NameOf = %s, want %s", name, want)
	}
	if parsed, err := archive.ParseName(name.String()); err != nil || parsed != name {
		t.Fatalf("ParseName(%s) = %v, %v", name, parsed, err)
	}
	m.Version = "v2.0.0-rc.1-3-g4de06ac+dirty/build"
	m.Trigger = archive.BeforeHandover
	long := archive.NameOf(m)
	if len(long.String()) > 80 || strings.ContainsAny(long.String(), "+/") {
		t.Fatalf("NameOf = %s, want at most 80 characters of the pattern", long)
	}
	if _, err := archive.ParseName(long.String()); err != nil {
		t.Fatalf("ParseName refused NameOf's own %s: %v", long, err)
	}
	for _, s := range []string{
		"",
		"..",
		"../slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age",
		"backups/slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age",
		`backups\slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age`,
		"slipstream-20261114T040012Z-2.1.0/../x-scheduled-7k3m.tar.zst.age",
		"slipstream-20261114T040012Z-2.1.0-nightly-7k3m.tar.zst.age",
		"slipstream-20261114T040012Z-2.1.0-scheduled-7k3u.tar.zst.age",
		"slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age.part",
		"slipstream-20261114T040012Z-2.1.0 -scheduled-7k3m.tar.zst.age",
		"slipstream-20261114T040012Z-" + strings.Repeat("9", 20) + "-before-handover-7k3m.tar.zst.age",
	} {
		if _, err := archive.ParseName(s); err == nil {
			t.Errorf("ParseName(%q) accepted it", s)
		}
	}
}

func TestAPassphraseIsTenWordsAndNeverPrints(t *testing.T) {
	p := archive.NewPassphrase(bytes.NewReader(bytes.Repeat([]byte{0xff, 0xff, 0x00, 0x00}, 5)))
	if want := "zoo abandon zoo abandon zoo abandon zoo abandon zoo abandon"; p.Reveal() != want {
		t.Fatalf("NewPassphrase = %q, want %q", p.Reveal(), want)
	}
	typed, err := archive.ParsePassphrase("  Zoo ABANDON\tzoo  abandon zoo abandon zoo abandon zoo\n abandon ")
	if err != nil || typed != p {
		t.Fatalf("ParsePassphrase = %q, %v; want it to match the generated one", typed.Reveal(), err)
	}
	if _, err := archive.ParsePassphrase(" \t"); err == nil {
		t.Fatal("ParsePassphrase accepted a blank passphrase")
	}
	var logged bytes.Buffer
	slog.New(slog.NewTextHandler(&logged, nil)).Info("taken", "pass", p)
	for _, s := range []string{fmt.Sprintf("%+v %#v %s", p, p, p), logged.String()} {
		if strings.Contains(s, "zoo") {
			t.Fatalf("the passphrase printed: %s", s)
		}
	}
}
