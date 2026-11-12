package storage_test

import (
	"encoding/json"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

// TestNewUUIDSortsAfterEveryEarlierOne mints an id at a clock behind the
// one an earlier id was minted at, as after the system clock steps back
// or beside a synctest bubble whose fake clock runs ahead: the later id
// still sorts after the earlier.
func TestNewUUIDSortsAfterEveryEarlierOne(t *testing.T) {
	mint, ahead := make(chan struct{}), make(chan storage.UUID)
	go func() {
		<-mint
		ahead <- storage.NewUUID()
	}()
	synctest.Test(t, func(t *testing.T) {
		first := storage.NewUUID()
		mint <- struct{}{}
		<-ahead
		if second := storage.NewUUID(); first.Compare(second) >= 0 {
			t.Fatalf("NewUUID minted %s after %s, want it to sort after", second, first)
		}
	})
}

func TestIDsAndTimesInJSONReadAsTheirValues(t *testing.T) {
	type row struct {
		ID    storage.UUID   `json:"id"`
		IDs   []storage.UUID `json:"ids"`
		At    storage.Time   `json:"at"`
		Never storage.Time   `json:"never,omitzero"`
	}
	in := row{ID: storage.NewUUID(), IDs: []storage.UUID{storage.NewUUID()}, At: storage.FromTime(time.Date(2026, 11, 12, 3, 4, 5, 6000, time.UTC))}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"` + in.ID.String() + `","ids":["` + in.IDs[0].String() + `"],"at":"2026-11-12T03:04:05.000006Z"}`
	if string(b) != want {
		t.Fatalf("json.Marshal = %s, want %s", b, want)
	}
	var out row
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != in.ID || !slices.Equal(out.IDs, in.IDs) || out.At != in.At || !out.Never.IsZero() {
		t.Errorf("json.Unmarshal = %+v, want %+v", out, in)
	}
}
