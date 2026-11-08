package imports

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

func (w *world) record(t *testing.T, root roots.ID, c Comparison) {
	t.Helper()
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return record(ctx, tx, root, c, storage.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (w *world) guard(t *testing.T) scanning.Guard {
	t.Helper()
	var g scanning.Guard
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		g, err = guardOf(ctx, r, w.root.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func (w *world) line(t *testing.T) (Line, bool) {
	t.Helper()
	lines, err := w.im.Lines(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	l, ok := lines[w.root.ID]
	return l, ok
}

func TestTheScanLineSaysWhatTheLastComparisonFound(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	if l, ok := w.line(t); ok {
		t.Fatalf("a root never compared has the line %+v, want none", l)
	}
	for _, c := range []struct {
		c    Comparison
		want string
	}{
		{Comparison{Found: 3, Gone: 1}, "It found 3 new files and 1 gone file."},
		{Comparison{Found: 1204}, "It found 1,204 new files."},
		{Comparison{Found: 1, Gone: 2, Returned: 3}, "It found 1 new file, 2 gone files and 3 returned files."},
		{Comparison{}, "Nothing changed."},
		{Comparison{Gone: 1912, Stopped: &scanning.Stopped{Gone: 1912, Of: 2000}}, "Library scan stopped because 1,912 of 2,000 files are gone. Check the disk or share it is on, then scan again."},
		{Comparison{Stopped: &scanning.Stopped{Gone: 1, Of: 3}}, "Library scan stopped because 1 of 3 files is gone. Check the disk or share it is on, then scan again."},
	} {
		c.c.Place = w.root.Place
		w.record(t, w.root.ID, c.c)
		l, ok := w.line(t)
		if !ok || l.Summary != c.want || l.ComparedAt.IsZero() || (l.Stopped != nil) != (c.c.Stopped != nil) {
			t.Errorf("after %+v the line reads %+v, want %q", c.c, l, c.want)
		}
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place + 1, Found: 3})
	if l, ok := w.line(t); ok {
		t.Errorf("a root compared only at another place has the line %+v, want none", l)
	}
	w.steps = map[tasks.UnitKey]tasks.Step{unitKey(w.root.ID): {Phase: string(Probing), Done: 12, Total: new(40)}}
	l, ok := w.line(t)
	if !ok || l.Running == nil || *l.Running != (Running{Run: 7, Phase: Probing, Done: 12, Total: l.Running.Total}) || *l.Running.Total != 40 {
		t.Errorf("while its unit probes the line reads %+v, want its run and step", l.Running)
	}
}

func TestMarkThemGoneConfirmsTheStoppedCountForTheNextComparison(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	if err := w.im.confirmGone(ctx, w.root.ID); !errors.Is(err, ErrNotStopped) {
		t.Fatalf("Mark them gone on a root never compared = %v, want ErrNotStopped", err)
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Found: 2})
	if err := w.im.confirmGone(ctx, w.root.ID); !errors.Is(err, ErrNotStopped) {
		t.Fatalf("Mark them gone after a comparison that didn't stop = %v, want ErrNotStopped", err)
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Stopped: &scanning.Stopped{Gone: 1912, Of: 2000}})
	if g := w.guard(t); g != (scanning.Guard{Share: 25}) {
		t.Fatalf("the guard before Mark them gone = %+v, want the stop share alone", g)
	}
	if err := w.im.confirmGone(ctx, w.root.ID); err != nil {
		t.Fatal(err)
	}
	if g := w.guard(t); g.Confirmed != 1912 {
		t.Fatalf("the guard after Mark them gone = %+v, want it to pass up to the 1,912 confirmed", g)
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Stopped: &scanning.Stopped{Gone: 1950, Of: 2000}, Consumed: 1000})
	if g := w.guard(t); g.Confirmed != 1912 {
		t.Errorf("a comparison that ran with another count left the guard %+v, want the 1,912 confirmed since", g)
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Gone: 1912, Consumed: 1912})
	if g := w.guard(t); g.Confirmed != 0 {
		t.Errorf("the comparison that ran with the confirmed count left the guard %+v, want it consumed", g)
	}
	if err := w.im.confirmGone(ctx, w.root.ID); !errors.Is(err, ErrNotStopped) {
		t.Errorf("Mark them gone after the comparison passed = %v, want ErrNotStopped", err)
	}
}

func TestSettingsStoreTheStopShareAndProbes(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	ctx := t.Context()
	for _, d := range []setting.Count{StopShare, Probes} {
		if err := d.Check(); err != nil {
			t.Error(err)
		}
	}
	panel := w.im.Settings()
	share, probes := countFields(t, panel)
	if share.Key != "library.scan.stopShare" || share.Value != 25 || share.Min != 1 || share.Max != 100 || share.Unit.Many != "%" {
		t.Errorf("the first field = %+v, want library.scan.stopShare at 25%% of 1 to 100", share)
	}
	if probes.Key != "library.scan.probes" || probes.Value != 2 || probes.Min != 1 || probes.Max != 8 {
		t.Errorf("the second field = %+v, want library.scan.probes at 2 of 1 to 8", probes)
	}
	if err := panel.Apply(ctx, setting.Patch{StopShare.Key: json.RawMessage(`40`), Probes.Key: json.RawMessage(`4`)}); err != nil {
		t.Fatal(err)
	}
	if g := w.guard(t); g.Share != 40 {
		t.Errorf("the guard after setting the stop share to 40 = %+v", g)
	}
	var rangeErr *setting.RangeError
	if err := panel.Apply(ctx, setting.Patch{Probes.Key: json.RawMessage(`9`), StopShare.Key: json.RawMessage(`10`)}); !errors.As(err, &rangeErr) || rangeErr.Key != Probes.Key {
		t.Errorf("Apply of 9 probes = %v, want a RangeError on probes", err)
	}
	var unknown *setting.UnknownKeyError
	if err := panel.Apply(ctx, setting.Patch{"library.scan.cadence": json.RawMessage(`1`)}); !errors.As(err, &unknown) {
		t.Errorf("Apply of a key the panel has no setting for = %v, want an UnknownKeyError", err)
	}
	if err := panel.Apply(ctx, setting.Patch{StopShare.Key: json.RawMessage(`null`)}); err != nil {
		t.Fatal(err)
	}
	if share, probes := countFields(t, panel); share.Value != 25 || probes.Value != 4 {
		t.Errorf("after the refused patch and a reset the fields read %d%% and %d probes, want 25%% and 4", share.Value, probes.Value)
	}
}

func countFields(t *testing.T, panel setting.Panel) (share, probes setting.CountField) {
	t.Helper()
	fields, err := panel.Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	share, okShare := fields[0].(setting.CountField)
	probes, okProbes := fields[1].(setting.CountField)
	if len(fields) != 2 || !okShare || !okProbes {
		t.Fatalf("the panel's fields = %+v, want two counts", fields)
	}
	return share, probes
}

func TestASkippedUnitSaysWhyItsRootIsntReady(t *testing.T) {
	for s, want := range map[roots.State]string{
		roots.Checking:  "Waiting for Media server to check the folder.",
		roots.Unmounted: "The folder is there but looks unmounted. Plug in or mount the disk.",
	} {
		if got := skipCause(s, "Media server"); got != want {
			t.Errorf("skipCause(%s) = %q, want %q", s, got, want)
		}
	}
}
