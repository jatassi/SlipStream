package storage_test

import (
	"testing"
	"testing/synctest"

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
