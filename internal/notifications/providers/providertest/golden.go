package providertest

import (
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/notifications/outbound"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

var update = flag.Bool("update", false, "rewrite the provider golden files")

// Endpoints stand in for the fixed hosts. A dump drops the host, so a
// golden reads the same for the provider's own host.
var Endpoints = providers.Endpoints{
	Telegram:   endpoint.MustParseURL("https://telegram.example"),
	Pushover:   endpoint.MustParseURL("https://pushover.example"),
	Pushbullet: endpoint.MustParseURL("https://pushbullet.example"),
	Ntfy:       endpoint.MustParseURL("https://ntfy.example"),
}

// Case is one connection's settings and opened secrets, which every
// fixture is built with.
type Case struct {
	Settings providers.Values
	Secrets  providers.Values
}

// Payload is f as p builds it for c to e, with the fixture poster when
// poster is set.
func Payload(p providers.Provider, c Case, f Fixture, poster bool, e providers.Endpoints) providers.Payload {
	out := providers.Payload{Kind: f.Kind, At: At, Message: f.Message, Settings: c.Settings, Secrets: c.Secrets, Endpoints: e}
	if poster {
		out.Poster = PosterFor(p)
	}
	return out
}

// Name is a golden file's name for f: its kind, then -poster when it
// carries one.
func Name(f Fixture, poster bool) string {
	if poster {
		return string(f.Kind) + "-poster"
	}
	return string(f.Kind)
}

// Golden builds p for c from every fixture, with and without a poster when
// both the fixture and p take one, and compares each dump with
// testdata/<id>/<name>.golden. go test -update rewrites them.
func Golden(t *testing.T, p providers.Provider, c Case) {
	t.Helper()
	for _, f := range Fixtures {
		for _, poster := range []bool{false, true} {
			if poster && (!f.Poster || p.Image == providers.NoImage) {
				continue
			}
			t.Run(Name(f, poster), func(t *testing.T) {
				out, err := p.Build(Payload(p, c, f, poster, Endpoints))
				if err != nil {
					t.Fatal(err)
				}
				Compare(t, filepath.Join("testdata", string(p.ID), Name(f, poster)+".golden"), outbound.DumpOutbound(out, Masks(c.Secrets)))
			})
		}
	}
}

// Compare checks got against the golden file at path, or rewrites it
// under -update.
func Compare(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // a golden file is meant to be read
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // a golden file under the test's testdata
	if err != nil {
		t.Fatalf("%v; run go test -update to write it", err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("%s differs; run go test -update and review the diff\n--- got\n%s\n--- want\n%s", path, got, want)
	}
}

// Masks are the strings of secrets a dump replaces: each value, each item
// of a list and each header row's value. Of a URL, they are its last path
// segment, its query values and its password, where a webhook URL keeps
// its token, so the dump still shows the route.
func Masks(secrets providers.Values) []string {
	var out []string
	for _, raw := range secrets {
		var s string
		var list []string
		var rows []providers.Header
		switch {
		case json.Unmarshal(raw, &s) == nil:
			out = append(out, secretParts(s)...)
		case json.Unmarshal(raw, &list) == nil:
			out = append(out, list...)
		case json.Unmarshal(raw, &rows) == nil:
			for _, h := range rows {
				out = append(out, h.Value)
			}
		}
	}
	return out
}

func secretParts(s string) []string {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return []string{s}
	}
	var out []string
	if i := strings.LastIndex(u.Path, "/"); i >= 0 && i < len(u.Path)-1 {
		out = append(out, u.Path[i+1:])
	}
	for _, vs := range u.Query() {
		out = append(out, vs...)
	}
	if pw, ok := u.User.Password(); ok {
		out = append(out, pw)
	}
	return out
}
