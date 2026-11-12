package grabs

import (
	"testing"

	"github.com/jatassi/slipstream/internal/works"
)

// TestAnImportReadsItsProgressInWords counts a Grab's Import in its Content
// Type's Item noun while each planned file holds its own Item, and in
// files once one doesn't.
func TestAnImportReadsItsProgressInWords(t *testing.T) {
	episode := works.Noun{One: "Episode", Many: "Episodes"}
	for _, c := range []struct {
		im   ImportOf
		want string
	}{
		{ImportOf{Reading: 10}, "Reading 10 files"},
		{ImportOf{Reading: 1}, "Reading 1 file"},
		{ImportOf{Planned: 10, Placed: 6, OneItemEach: true}, "Importing 6 of 10 episodes"},
		{ImportOf{Planned: 1, OneItemEach: true}, "Importing 0 of 1 episode"},
		{ImportOf{Planned: 9, Placed: 2}, "Importing 2 of 9 files"},
		{ImportOf{Planned: 10, Placed: 10, OneItemEach: true, Finished: true}, "Imported 10 of 10 episodes"},
		{ImportOf{Finished: true}, "Imported no files"},
	} {
		if got := c.im.words(episode); got != c.want {
			t.Errorf("%+v reads %q, want %q", c.im, got, c.want)
		}
	}
}
