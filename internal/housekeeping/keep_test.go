package housekeeping_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestKeepHistoryForIsHousekeepingsTuning(t *testing.T) {
	lib := storagetest.Library(t, history.Schema())
	h := history.New(history.Config{Library: lib, Catalog: entry.NewCatalog(), Log: slog.New(slog.DiscardHandler)})
	tuning := housekeeping.Task(housekeeping.Config{Keep: h}).Tuning
	days := func(n int64) setting.Patch {
		return setting.Patch{history.KeepFor.Key: json.RawMessage(strconv.FormatInt(n*86400, 10))}
	}
	var refused *setting.RangeError
	if err := tuning.Apply(t.Context(), days(20)); !errors.As(err, &refused) {
		t.Fatalf("20 days = %v, want a range refusal", err)
	}
	if err := tuning.Apply(t.Context(), days(45)); err != nil {
		t.Fatal(err)
	}
	fields, err := tuning.Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if keep, ok := fields[0].(setting.DurationField); len(fields) != 1 || !ok || keep.Label != "Keep history for" || keep.Value != 45*86400 {
		t.Fatalf("the Tuning lists %+v, want Keep history for alone, at 45 days", fields)
	}
	if keep, err := h.Keep(t.Context()); err != nil || keep != 45*24*time.Hour {
		t.Fatalf("Keep = %v, %v, want 45 days", keep, err)
	}
}
