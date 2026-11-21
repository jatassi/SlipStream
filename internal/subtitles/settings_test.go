package subtitles_test

import (
	"encoding/json"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// values is each field's key and value as JSON.
func values(t *testing.T, p setting.Panel) map[string]string {
	t.Helper()
	fields, err := p.Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range fields {
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatal(err)
		}
		out[v.Key] = string(v.Value)
	}
	return out
}

func TestTheSettingsRoundTripAndResetToTheirDefaults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Subs.Settings()
		got := values(t, p)
		if got["subtitles.sync.match"] != "30" || timing.DefaultMatchPercent != 30 || got["subtitles.sync.framerates"] != `"standard"` || got["subtitles.searchOn"] != "true" {
			t.Fatalf("the defaults read %v", got)
		}
		if _, shown := got["subtitles.reportGap"]; shown {
			t.Fatal("the report gap shows before S47 (Subtitle reports) supplies reports")
		}
		patch := setting.Patch{
			"subtitles.searchOn": json.RawMessage(`false`), "subtitles.sync.match": json.RawMessage(`55`),
			"subtitles.sync.framerates": json.RawMessage(`"range"`), "subtitles.longestGap": json.RawMessage(`172800`),
			"subtitles.reportGap": json.RawMessage(`7200`),
		}
		if err := p.Apply(t.Context(), patch); err != nil {
			t.Fatal(err)
		}
		got = values(t, p)
		if got["subtitles.searchOn"] != "false" || got["subtitles.sync.match"] != "55" || got["subtitles.sync.framerates"] != `"range"` || got["subtitles.longestGap"] != "172800" {
			t.Fatalf("after the patch the settings read %v", got)
		}
		if err := p.Apply(t.Context(), setting.Patch{"subtitles.sync.match": json.RawMessage(`null`)}); err != nil {
			t.Fatal(err)
		}
		if got := values(t, p)["subtitles.sync.match"]; got != "30" {
			t.Fatalf("a reset match reads %s, want the default 30", got)
		}
		if n := len(w.ProvidersChanged.All()); n != 2 {
			t.Fatalf("two saves published subtitles.providers-changed %d times", n)
		}
	})
}

func TestTheSettingsRefuseAGapUnderTheIntervalAndValuesOutOfRange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Subs.Settings()
		err := p.Apply(t.Context(), setting.Patch{"subtitles.longestGap": json.RawMessage(`3600`)})
		var r *setting.RangeError
		if !errors.As(err, &r) {
			t.Fatalf("a 1 hour gap = %v, want the setting's own range refusal", err)
		}
		short := &subtitles.GapTooShortError{Key: subtitles.LongestGap.Key, Gap: 4 * time.Hour, Interval: 8 * time.Hour}
		if got, want := short.Words(), "Use 8 hours or longer. Subtitle search runs every 8 hours."; got != want {
			t.Fatalf("the gap key's words = %q, want %q", got, want)
		}
		long := &subtitles.GapTooShortError{Key: subtitles.Every.Key, Gap: 7 * 24 * time.Hour, Interval: 8 * 24 * time.Hour}
		if got, want := long.Words(), "Use 7 days or shorter. A subtitle that keeps coming up empty is searched for at least every 7 days."; got != want {
			t.Fatalf("the cadence key's words = %q, want %q", got, want)
		}
		if err := p.Apply(t.Context(), setting.Patch{"subtitles.sync.match": json.RawMessage(`5`)}); !errors.As(err, &r) {
			t.Fatalf("a 5%% match = %v, want a range refusal", err)
		}
		if err := p.Apply(t.Context(), setting.Patch{subtitles.Every.Key: json.RawMessage(`43200`)}); err != nil {
			t.Fatal(err)
		}
		if got := values(t, p)[subtitles.Every.Key]; got != "43200" {
			t.Fatalf("a 12 hour cadence reads %s", got)
		}
		var u *setting.UnknownKeyError
		if err := p.Apply(t.Context(), setting.Patch{"subtitles.mapSlots": json.RawMessage(`2`)}); !errors.As(err, &u) {
			t.Fatalf("a Node's setting on the section = %v, want an unknown key", err)
		}
	})
}

func TestANodeKeepsItsAudioScanSlots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Subs.NodeSettings(w.Node)
		if got := values(t, p)["subtitles.mapSlots"]; got != "1" {
			t.Fatalf("a Node's slots read %s, want the default 1", got)
		}
		if err := p.Apply(t.Context(), setting.Patch{"subtitles.mapSlots": json.RawMessage(`4`)}); err != nil {
			t.Fatal(err)
		}
		if got := values(t, p)["subtitles.mapSlots"]; got != "4" {
			t.Fatalf("after setting 4 the slots read %s", got)
		}
		if err := p.Apply(t.Context(), setting.Patch{"subtitles.mapSlots": json.RawMessage(`null`)}); err != nil {
			t.Fatal(err)
		}
		if rows := query(t, w, `SELECT hex(node_id) FROM subtitles_node`); len(rows) != 0 {
			t.Fatalf("a reset Node with no fault keeps its row %v", rows)
		}
		w.Holders(false)
		var not *subtitles.NotStorageError
		if _, err := p.Fields(t.Context()); !errors.As(err, &not) {
			t.Fatalf("a Node without the Storage Role = %v, want NotStorageError", err)
		}
	})
}
