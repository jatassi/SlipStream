package grabs

import (
	"context"
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ImportOf is a Grab's Import as Activity shows it. Planned counts its
// files that are or were sending, placing, placed or waiting; Reading,
// Placed, Sending and Waiting count those at each.
type ImportOf struct {
	ID                                         storage.UUID
	Reading, Planned, Placed, Sending, Waiting int
	// OneItemEach is whether each planned file holds one Item and no two
	// hold the same one, so its count is a count of Items.
	OneItemEach bool
	Finished    bool
}

// Imports reads Grabs' Imports, which imports implements.
type Imports interface {
	// Of reads the Import of each of ids that has one, its unfinished one
	// first, in r.
	Of(ctx context.Context, r storage.Reader[file], ids []GrabID) (map[GrabID]ImportOf, error)
}

// ImportView is a Grab's Import with its progress in words: "Importing 6
// of 10 episodes".
type ImportView struct {
	ImportOf
	Words string
}

func (im ImportOf) words(item works.Noun) string {
	noun := works.Noun{One: "file", Many: "files"}
	if im.OneItemEach {
		noun = works.Noun{One: strings.ToLower(item.One), Many: strings.ToLower(item.Many)}
	}
	switch {
	case im.Planned == 0 && !im.Finished:
		return "Reading " + plural(im.Reading, "1 file", "%d files")
	case im.Planned == 0:
		return "Imported no files"
	}
	verb, of := "Importing", noun.Many
	if im.Finished {
		verb = "Imported"
	}
	if im.Planned == 1 {
		of = noun.One
	}
	return fmt.Sprintf("%s %d of %d %s", verb, im.Placed, im.Planned, of)
}

// hasImport reports whether g got as far as its Import.
func (g Grab) hasImport() bool {
	return g.Stage == Importing || g.Stage == Imported || g.Failed != nil && g.Failed.Stage == Importing
}

// importsOf is the Import of each of grabs that has one, with its words in
// the Item noun of its Work's Content Type.
func (gr *Grabs) importsOf(ctx context.Context, r storage.Reader[file], grabs []Grab, types map[works.WorkID]works.TypeID) (map[GrabID]ImportView, error) {
	var ids []GrabID
	for _, g := range grabs {
		if g.hasImport() {
			ids = append(ids, g.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	of, err := gr.cfg.Imports.Of(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[GrabID]ImportView, len(of))
	for _, g := range grabs {
		im, ok := of[g.ID]
		if !ok {
			continue
		}
		d, _ := gr.cfg.Works.Describe(types[g.WorkID])
		out[g.ID] = ImportView{ImportOf: im, Words: im.words(d.Item)}
	}
	return out, nil
}
