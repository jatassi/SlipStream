package files_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// fileDraft is a draft of the Fixture's file template.
func fileDraft(template string) *files.Draft {
	return &files.Draft{Type: contenttypetest.ID, Patch: setting.Patch{string(contenttypetest.ID) + ".naming.file.standard": json.RawMessage(template)}}
}

// bracketed is the Fixture's file template with its Target in brackets.
const bracketed = `"{Shelf} - {Thing} - {Edition Tag} [{Target}]"`

type renamed struct{ from, to string }

func namesOf(t *testing.T, svc *files.Files, draft *files.Draft, w works.WorkID) []renamed {
	t.Helper()
	nm, err := svc.NamesOf(t.Context(), admin, w, draft)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]renamed, len(nm))
	for i, n := range nm {
		out[i] = renamed{n.From, n.To}
	}
	return out
}

func TestAWorksNamesListEveryLiveFileUnderADraftAndWriteNothing(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1 := s.work(t, "w1", "Shelf w1")
	s.found(t, w1, 0, "Shelf w1/Shelf w1 - Scarf - Web.mkv")
	s.found(t, w1, 1, "Loft/m.mkv")
	before := s.l.rows(t)

	stored := namesOf(t, svc, nil, w1.ID)
	if want := []renamed{{"Shelf w1/Shelf w1 - Scarf - Web.mkv", "Shelf w1/Shelf w1 - Scarf - Web.mkv"}, {"Loft/m.mkv", "Shelf w1/Shelf w1 - Mittens - Web.mkv"}}; !slices.Equal(stored, want) {
		t.Fatalf("names under the stored settings = %v, want the matching file listed as it is and the found one moved: %v", stored, want)
	}
	drafted := namesOf(t, svc, fileDraft(bracketed), w1.ID)
	if want := []renamed{{"Shelf w1/Shelf w1 - Scarf - Web.mkv", "Shelf w1/Shelf w1 - Scarf - [Web].mkv"}, {"Loft/m.mkv", "Shelf w1/Shelf w1 - Mittens - [Web].mkv"}}; !slices.Equal(drafted, want) {
		t.Fatalf("names under the draft = %v, want %v", drafted, want)
	}
	_, err := svc.NamesOf(t.Context(), admin, w1.ID, fileDraft(`"{Shelf} - {Thing}"`))
	if refused, ok := errors.AsType[*naming.RefusedError](err); !ok || len(refused.Problems) == 0 {
		t.Fatalf("names under a template without {Target} = %v, want the save check's refusal", err)
	}
	if after := s.l.rows(t); !maps.Equal(after, before) {
		t.Fatalf("files after naming = %v, want them as they were: %v", after, before)
	}
}

func TestAnImaginedFileIsNamedUnderTheDraftAndWritesNothing(t *testing.T) {
	s := newAttic(t)
	svc := s.l.files(files.NoGrabs{})
	w1 := s.work(t, "w1", "Shelf w1")
	s.found(t, w1, 0, "Shelf w1/Shelf w1 - Scarf - Web.mkv")
	before := s.l.rows(t)
	in := files.Batch{Work: w1.ID, Root: s.l.root.ID, Candidates: []files.Candidate{{
		Items: []works.ItemID{w1.Items[1].ID}, Cut: release.Standard, Holding: files.HoldingOf(storage.NewUUID(), ".mkv"),
		Quality: release.WEBDL1080p, SourceName: "mittens.mkv",
	}}}

	var stored, drafted string
	err := s.l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if stored, err = svc.Imagine(ctx, r, in, nil); err != nil {
			return err
		}
		drafted, err = svc.Imagine(ctx, r, in, fileDraft(bracketed))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if stored != "Shelf w1/Shelf w1 - Mittens - Web.mkv" || drafted != "Shelf w1/Shelf w1 - Mittens - [Web].mkv" {
		t.Fatalf("imagined paths = %q and %q, want the stored template's and the draft's", stored, drafted)
	}
	if after := s.l.rows(t); !maps.Equal(after, before) {
		t.Fatalf("files after imagining = %v, want them as they were: %v", after, before)
	}
}
