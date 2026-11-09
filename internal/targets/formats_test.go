package targets_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type shelf struct {
	lib   *storage.DB[storage.Library]
	goals *targets.Targets
	works []works.WorkID
}

// newShelf opens targets over n cinema Works, which want the default
// Profile's starter Target.
func newShelf(t *testing.T, n int) *shelf {
	t.Helper()
	ctx := t.Context()
	s := &shelf{lib: storagetest.Library(t, works.Schema(), targets.Schema())}
	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i := range n {
			w, err := seedWork(ctx, tx, "cinema", string(rune('a'+i)))
			if err != nil {
				return err
			}
			s.works = append(s.works, w)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s.goals = must[*targets.Targets](t)(targets.Open(ctx, targets.Config{
		Library: s.lib, Works: described{"cinema": "Cinema"}, Types: facets("cinema"), Changed: discard{}, Formats: formats.DefaultTimings(),
	}))
	return s
}

func (s *shelf) apply(t *testing.T, c targets.Change) error {
	t.Helper()
	return s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		known, err := s.goals.CutsIn(ctx, tx)
		if err != nil {
			return err
		}
		return targets.Apply(ctx, tx, known, c)
	})
}

func (s *shelf) touches(t *testing.T, c targets.Change) []works.WorkID {
	t.Helper()
	var got targets.Touched
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = s.goals.Touches(ctx, r, c)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got.Works
}

func (s *shelf) format(t *testing.T, id formats.ID) targets.StoredFormat {
	t.Helper()
	return must[targets.StoredFormat](t)(s.goals.GetFormat(t.Context(), identity.Admin{}, id))
}

func titled(name, pattern string) formats.Format {
	fields, err := json.Marshal(map[string]string{"value": pattern})
	if err != nil {
		panic(err)
	}
	return formats.Format{ID: targets.NewFormatID(), Name: name, Conditions: []formats.Condition{{Name: name, Kind: formats.ReleaseTitle, Fields: fields}}}
}

// scoring is the Target Scored, which scores formats as scores do.
func scoring(scores map[formats.ID]int) targets.Spec {
	return targets.Spec{
		Name: "Scored", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
		Scores: targets.Scores{Formats: scores, Increment: 1},
	}
}

func TestAFormatChangeTouchesOnlyTheWorksWhoseProfilesHoldATargetScoringIt(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 3)
	hdr, remux := titled("HDR", `\bHDR\b`), titled("Remux", `\bRemux\b`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr, remux}}); err != nil {
		t.Fatal(err)
	}
	scored := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, scoring(map[formats.ID]int{hdr.ID: 10})))
	p := must[targets.Profile](t)(s.goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{
		Name: "Scoring", ContentType: "cinema", Entries: []targets.Entry{{Target: scored.ID, Sourcing: targets.Download}}, Editions: targets.AnyCut(),
	}))
	if err := s.apply(t, targets.SetProfiles{Works: s.works[:2], Profile: p.ID}); err != nil {
		t.Fatal(err)
	}
	want := slices.SortedFunc(slices.Values(s.works[:2]), func(a, b works.WorkID) int { return a.Compare(b.UUID) })

	for _, c := range []struct {
		name   string
		change targets.Change
		want   []works.WorkID
	}{
		{"an edit to a format no Target scores", targets.UpdateFormat{ID: remux.ID, Format: titled("Remux", `Remux`)}, nil},
		{"deleting a format no Target scores", targets.DeleteFormat{ID: remux.ID}, nil},
		{"adding a format", targets.AddFormats{Dialect: formats.Radarr, Add: []formats.Format{titled("DV", `\bDV\b`)}}, nil},
		{"replacing a format no Target scores", targets.AddFormats{Dialect: formats.Radarr, Replace: map[formats.ID]formats.Format{remux.ID: titled("Remux", `REMUX`)}}, nil},
		{"an edit to a scored format", targets.UpdateFormat{ID: hdr.ID, Format: titled("HDR", `HDR10`)}, want},
		{"deleting a scored format", targets.DeleteFormat{ID: hdr.ID}, want},
		{"replacing a scored format", targets.AddFormats{Dialect: formats.Radarr, Replace: map[formats.ID]formats.Format{hdr.ID: titled("HDR", `HDR`)}}, want},
	} {
		if got := s.touches(t, c.change); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s touches %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAFormatKeepsItsPastedDocumentAndCountsTimeoutsUntilItsNextSave(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 0)
	pasted := titled("HDR", `\bHDR\b`)
	pasted.Source = []byte(`{"trash_id":"abc","name":"HDR"}`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.Radarr, Add: []formats.Format{pasted}}); err != nil {
		t.Fatal(err)
	}
	scored := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, scoring(map[formats.ID]int{pasted.ID: 25})))

	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return targets.AddTimeouts(ctx, tx, []formats.Result{
			{Timeouts: []formats.Timeout{{Format: pasted.ID, Condition: 0}, {Format: pasted.ID, Condition: 1}}},
			{Timeouts: []formats.Timeout{{Format: pasted.ID, Condition: 0}}},
			{},
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	got := s.format(t, pasted.ID)
	if got.Timeouts != 2 || got.Revision != 1 || got.Dialect != formats.Radarr || !bytes.Equal(got.Source, pasted.Source) {
		t.Fatalf("after a search = %d timeouts at revision %d in %s from %q, want 2 timeouts at revision 1 in radarr from the pasted document", got.Timeouts, got.Revision, got.Dialect, got.Source)
	}
	if !reflect.DeepEqual(got.ScoredBy, []targets.TargetRef{{ID: scored.ID, Name: "Scored"}}) {
		t.Fatalf("ScoredBy = %+v, want Scored", got.ScoredBy)
	}

	edited := titled("HDR10", `\bHDR10\b`)
	edited.Dialect, edited.Source = formats.Sonarr, []byte(`{}`)
	if err := s.apply(t, targets.UpdateFormat{ID: pasted.ID, Format: edited}); err != nil {
		t.Fatal(err)
	}
	got = s.format(t, pasted.ID)
	if got.Name != "HDR10" || !reflect.DeepEqual(got.Conditions, edited.Conditions) || got.Timeouts != 0 || got.Revision != 2 {
		t.Fatalf("after an edit = %+v, want HDR10's conditions at revision 2 with no timeouts", got)
	}
	if got.Dialect != formats.Radarr || !bytes.Equal(got.Source, pasted.Source) {
		t.Fatalf("an edit changed the dialect to %s or the document to %q, want both kept", got.Dialect, got.Source)
	}
}

func TestAFormatNameIsUniqueCaseInsensitivelyAndAPatternMustCompile(t *testing.T) {
	s := newShelf(t, 0)
	hdr := titled("HDR", `HDR`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		add  []formats.Format
	}{
		{"an add", []formats.Format{titled("hdr", `x`)}},
		{"two adds", []formats.Format{titled("DV", `x`), titled("dv", `y`)}},
	} {
		if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: c.add}); !errors.Is(err, targets.ErrFormatNameTaken) {
			t.Errorf("%s of a taken name = %v, want ErrFormatNameTaken", c.name, err)
		}
	}
	if err := s.apply(t, targets.UpdateFormat{ID: hdr.ID, Format: titled("HDR", `x`)}); err != nil {
		t.Fatalf("an edit keeping the format's own name = %v, want it saved", err)
	}
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Replace: map[formats.ID]formats.Format{hdr.ID: titled("SDR", `x`)}, Add: []formats.Format{titled("Hdr", `x`)}}); err != nil {
		t.Fatalf("an add taking the name a replace gives up = %v, want it saved", err)
	}
	broken := titled("Broken", `(HDR`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{broken}}); !isPatternError(err) {
		t.Fatalf("a pattern that doesn't compile = %v, want a *formats.PatternError", err)
	}
	if err := s.apply(t, targets.UpdateFormat{ID: targets.NewFormatID(), Format: titled("Gone", `x`)}); !errors.Is(err, targets.ErrNoFormat) {
		t.Fatalf("editing no format = %v, want ErrNoFormat", err)
	}
	if err := s.apply(t, targets.AddFormats{Dialect: formats.Sonarr, Replace: map[formats.ID]formats.Format{targets.NewFormatID(): titled("Gone", `x`)}}); !errors.Is(err, targets.ErrNoFormat) {
		t.Fatalf("replacing no format = %v, want ErrNoFormat", err)
	}
}

func isPatternError(err error) bool {
	_, ok := errors.AsType[*formats.PatternError](err)
	return ok
}

func TestDeletingAFormatDropsItsScoresAndAScoreMustNameAFormat(t *testing.T) {
	ctx := t.Context()
	s := newShelf(t, 0)
	hdr, dv := titled("HDR", `HDR`), titled("DV", `DV`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr, dv}}); err != nil {
		t.Fatal(err)
	}
	spec := scoring(map[formats.ID]int{hdr.ID: 10, dv.ID: -5, targets.NewFormatID(): 0})
	spec.Scores.Minimum, spec.Scores.UpgradeUntil, spec.Scores.Increment, spec.Propers = -20, 100, 5, targets.DontPrefer
	scored := must[targets.Target](t)(s.goals.CreateTarget(ctx, identity.Admin{}, spec))
	got := must[targets.Target](t)(s.goals.GetTarget(ctx, identity.Admin{}, scored.ID))
	want := targets.Scores{Formats: map[formats.ID]int{hdr.ID: 10, dv.ID: -5}, Minimum: -20, UpgradeUntil: 100, Increment: 5}
	if !reflect.DeepEqual(got.Scores, want) || got.Propers != targets.DontPrefer {
		t.Fatalf("stored %+v and %s, want %+v and dont_prefer, a 0 score left out", got.Scores, got.Propers, want)
	}

	if err := s.apply(t, targets.DeleteFormat{ID: hdr.ID}); err != nil {
		t.Fatal(err)
	}
	got = must[targets.Target](t)(s.goals.GetTarget(ctx, identity.Admin{}, scored.ID))
	if !reflect.DeepEqual(got.Scores.Formats, map[formats.ID]int{dv.ID: -5}) {
		t.Fatalf("after deleting HDR the Target scores %v, want DV alone", got.Scores.Formats)
	}
	if _, err := s.goals.GetFormat(ctx, identity.Admin{}, hdr.ID); !errors.Is(err, targets.ErrNoFormat) {
		t.Fatalf("reading a deleted format = %v, want ErrNoFormat", err)
	}
	if err := s.apply(t, targets.UpdateTarget{ID: scored.ID, Spec: scoring(map[formats.ID]int{hdr.ID: 10})}); !errors.Is(err, targets.ErrNoFormat) {
		t.Fatalf("scoring a deleted format = %v, want ErrNoFormat", err)
	}
	if err := s.apply(t, targets.DeleteFormat{ID: hdr.ID}); !errors.Is(err, targets.ErrNoFormat) {
		t.Fatalf("deleting a deleted format = %v, want ErrNoFormat", err)
	}
}

func TestFormatsInCompilesTheCatalogOncePerSave(t *testing.T) {
	s := newShelf(t, 0)
	hdr := titled("HDR", `\bHDR\b`)
	if err := s.apply(t, targets.AddFormats{Dialect: formats.SlipStream, Add: []formats.Format{hdr}}); err != nil {
		t.Fatal(err)
	}
	catalog := func() *formats.Catalog {
		var c *formats.Catalog
		err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			c, err = s.goals.FormatsIn(ctx, r)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	matches := func(c *formats.Catalog, title string) bool {
		return slices.Contains(c.Match(release.Read{Release: title}).Matched, hdr.ID)
	}
	first := catalog()
	if !matches(first, "Heat.1995.2160p.HDR.WEB-DL") {
		t.Fatal("the catalog doesn't match HDR in an HDR title")
	}
	if again := catalog(); again != first {
		t.Fatal("FormatsIn compiled the catalog again with no save between")
	}
	if err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return targets.AddTimeouts(ctx, tx, []formats.Result{{Timeouts: []formats.Timeout{{Format: hdr.ID}}}})
	}); err != nil {
		t.Fatal(err)
	}
	if again := catalog(); again != first {
		t.Fatal("FormatsIn compiled the catalog again after a timeout count, which changes no pattern")
	}
	if err := s.apply(t, targets.UpdateFormat{ID: hdr.ID, Format: titled("HDR", `\bHDR10\b`)}); err != nil {
		t.Fatal(err)
	}
	edited := catalog()
	if edited == first || matches(edited, "Heat.1995.2160p.HDR.WEB-DL") || !matches(edited, "Heat.1995.2160p.HDR10.WEB-DL") {
		t.Fatal("after an edit FormatsIn still matches the old pattern, want the edited one")
	}
}
