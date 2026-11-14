package imports

import (
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// TestAnImportCountsItsFilesForActivity counts the files that are or were
// sending, placing, placed or waiting as planned, the rest of a download
// apart, and a file of two Items, or two files of one Item, as no count of
// Items.
func TestAnImportCountsItsFilesForActivity(t *testing.T) {
	one, two := works.ItemID{UUID: storage.NewUUID()}, works.ItemID{UUID: storage.NewUUID()}
	file := func(o Outcome, items ...works.ItemID) File { return File{Outcome: o, Items: items} }
	for _, c := range []struct {
		name  string
		files []File
		want  grabs.ImportOf
	}{
		{"a pack mid import", []File{file(Placed, one), file(Sending, two), file(LeftOut), file(Pending)},
			grabs.ImportOf{Reading: 1, Planned: 2, Placed: 1, Sending: 1, OneItemEach: true}},
		{"a file waits", []File{file(Waiting, one), file(Placing, two), file(Declined, one)},
			grabs.ImportOf{Planned: 2, Waiting: 1, OneItemEach: true}},
		{"a file of two episodes", []File{file(Placed, one, two)}, grabs.ImportOf{Planned: 1, Placed: 1}},
		{"two files of one movie", []File{file(Placed, one), file(Placing, one)}, grabs.ImportOf{Planned: 2, Placed: 1}},
	} {
		if got := (Import{Files: c.files}).progress(); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

// TestAHeldCellNamesItsFurthestFilesStep names the step of the file that
// holds a cell and is nearest the library, and a probe hold the Admin must
// answer as waiting.
func TestAHeldCellNamesItsFurthestFilesStep(t *testing.T) {
	item := works.ItemID{UUID: storage.NewUUID()}
	k := files.CellKey{Item: item}
	filling := func(o Outcome) File { return File{Outcome: o, Items: []works.ItemID{item}, Fills: []Fill{{Cell: k}}} }
	offered := File{Outcome: Pending, Hold: &Hold{Code: ProbeTimeout, Attempts: 3}}
	for _, c := range []struct {
		name  string
		files []File
		want  files.Step
	}{
		{"a file reads", []File{{Outcome: Pending}}, files.StepReading},
		{"a probe timed out three times", []File{offered}, files.StepWaiting},
		{"a file waits confirm while another reads", []File{{Outcome: Pending}, {Outcome: Waiting, Reason: &Reason{Code: Confirm}}}, files.StepWaiting},
		{"a file sends", []File{{Outcome: Pending}, filling(Sending)}, files.StepSending},
		{"a file places", []File{filling(Sending), filling(Placing)}, files.StepPlacing},
		{"no file holds it", []File{filling(Placed), {Outcome: LeftOut}}, ""},
	} {
		if got := (Import{Files: c.files}).stepOn(k); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
