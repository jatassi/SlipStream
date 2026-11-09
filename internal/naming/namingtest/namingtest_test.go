package namingtest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/naming/namingtest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type namer struct{ d naming.Declaration }

func (n namer) Naming() naming.Declaration { return n.d }

func (namer) NamingValues(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
	return nil, nil
}

// recorder is a testing.TB that keeps what Conform reports.
type recorder struct {
	testing.TB
	errors []string
}

func (*recorder) Helper() {}

func (r *recorder) Error(args ...any) { r.errors = append(r.errors, fmt.Sprint(args...)) }

var shelf = naming.Declaration{
	Tokens: []naming.Token{{Name: "Shelf", Kind: naming.KindText, Scope: naming.WorkScope}},
	Folder: "{Shelf}",
	Files:  []naming.Variant{{Key: "standard", Label: "File", File: "{Shelf} - {Edition Tag} {Target}", Anchor: naming.FolderAnchor()}},
	Samples: []naming.Sample{
		{Label: "A shelf", Work: naming.Set{"Shelf": naming.Text("Top")}},
		{Label: "Nothing known", Missing: true, Work: naming.Set{"Shelf": naming.Text("Top")}},
	},
}

func TestConformPassesADeclarationThatKeepsEveryRule(t *testing.T) {
	r := &recorder{}
	namingtest.Conform(r, namer{shelf})
	if r.errors != nil {
		t.Errorf("Conform reported %q; want nothing", r.errors)
	}
}

func TestConformReportsEachBrokenRule(t *testing.T) {
	broken := shelf
	broken.Folder = "{Shelf} {Target}"
	broken.Samples = []naming.Sample{{Label: "A number", Work: naming.Set{"Shelf": naming.Number(3)}}}
	r := &recorder{}
	namingtest.Conform(r, namer{broken})
	want := []string{
		`sample "A number" has a value for "Shelf" of a kind the token doesn't hold`,
		"the default folder: Folder names can't use {Target}, because it changes with each upgrade. Remove it from the folder template.",
	}
	if len(r.errors) != len(want) || r.errors[0] != want[0] || r.errors[1] != want[1] {
		t.Errorf("Conform reported %q; want %q", r.errors, want)
	}
}
