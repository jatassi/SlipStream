package files

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAnImpactListsItsFirst500RenamesByWorkInTitleOrder(t *testing.T) {
	var (
		trees []works.Bare
		rs    []Renames
	)
	for _, w := range []struct {
		title, sortLabel string
		files            int
	}{{"The Zoo", "Zoo", 1}, {"An Attic", "Attic", 300}, {"Moor", "Moor", 300}} {
		tree := works.Bare{Summary: works.Summary{ID: works.WorkID{UUID: storage.NewUUID()}, Label: w.title, SortLabel: w.sortLabel}}
		trees = append(trees, tree)
		r := Renames{Work: tree.ID, Title: w.title}
		for n := range w.files {
			r.Rows = append(r.Rows, Row{Work: tree.ID, File: ID{UUID: storage.NewUUID()}, From: fmt.Sprintf("%d.mkv", n), To: fmt.Sprintf("%d - HD.mkv", n)})
		}
		rs = append(rs, r)
	}
	var got []string
	for _, r := range listed(rs, trees) {
		got = append(got, fmt.Sprintf("%s %d", r.Title, len(r.Rows)))
	}
	if want := []string{"An Attic 300", "Moor 200"}; !slices.Equal(got, want) {
		t.Fatalf("the Impact lists %v, want %v: 500 rows by sort title, the last Work cut", got, want)
	}
}
