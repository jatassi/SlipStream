package storage

import (
	"time"
	"unsafe"

	"modernc.org/libc"
	sqlite3 "modernc.org/sqlite/lib"
)

// SQLite sleeps through its VFS's xSleep while it waits out a lock or a
// WAL read racing a commit. modernc's libc sleeps on Go's timer on macOS
// and Windows, which a synctest bubble fakes, so synctest.Wait took a
// statement waiting to retry for a settled one. xSleep sleeps on the OS
// clock on every platform, as Linux's libc does.
func init() {
	tls := libc.NewTLS()
	defer tls.Close()
	p := sqlite3.Xsqlite3_vfs_find(tls, 0)
	vfs := *(**sqlite3.Tsqlite3_vfs)(unsafe.Pointer(&p)) //nolint:gosec // G103: SQLite's default VFS for a C call
	sleep := struct {
		f func(*libc.TLS, uintptr, int32) int32
	}{xSleep}
	vfs.FxSleep = *(*uintptr)(unsafe.Pointer(&sleep)) //nolint:gosec // G103: a Go func as a C function pointer
}

func xSleep(_ *libc.TLS, _ uintptr, microseconds int32) int32 {
	sleepOS(time.Duration(microseconds) * time.Microsecond)
	return microseconds
}
