package storage_test

import (
	"sync/atomic"
	"testing"
	"testing/synctest"

	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestSynctestWaitsForSQLiteWaitingOutALock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var slept atomic.Bool
		go func() {
			tls := libc.NewTLS()
			defer tls.Close()
			sqlite3.Xsqlite3_sleep(tls, 1)
			slept.Store(true)
		}()
		synctest.Wait()
		if !slept.Load() {
			t.Fatal("synctest.Wait returned while SQLite slept, so a statement waiting out a lock looked settled")
		}
	})
}
