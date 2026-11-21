package backups_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/storage"
)

func (w *world) replace(deleteOlder bool) backups.Entry {
	w.t.Helper()
	run, err := w.backups.Replace(w.t.Context(), storage.NewUUID(), deleteOlder)
	if err != nil {
		w.t.Fatal(err)
	}
	e, err := run.Wait(w.t.Context())
	if err != nil {
		w.t.Fatal(err)
	}
	return e
}

func (w *world) generations() []backups.Generation {
	w.t.Helper()
	v, err := w.eng.Vault(w.keys)
	if err != nil {
		w.t.Fatal(err)
	}
	return v.Generations()
}

func TestReplaceDeletesOlderBackupsOnceTheNewOnePasses(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	w.taken(archive.Scheduled)
	w.taken(archive.Manual)
	before, err := w.backups.Passphrase()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.backups.Saved(); err != nil {
		t.Fatal(err)
	}
	replaced := w.replace(true)
	if got, want := w.listed(), []string{replaced.Name.String() + " verified"}; !slices.Equal(got, want) {
		t.Fatalf("listed %q, want only the Backup of the new passphrase %q", got, want)
	}
	if got, want := w.files(), []string{replaced.Name.String()}; !slices.Equal(got, want) {
		t.Fatalf("the Backups folder holds %q, want %q", got, want)
	}
	if got := w.generations(); !slices.Equal(got, []backups.Generation{replaced.Generation}) {
		t.Fatalf("backup.key keeps generations %v, want only %d", got, replaced.Generation)
	}
	after, err := w.backups.Passphrase()
	if err != nil {
		t.Fatal(err)
	}
	if after.Reveal() == before.Reveal() {
		t.Fatal("the passphrase is the one it replaced")
	}
	if !w.ledger().PassphraseConfirmedAt.IsZero() {
		t.Fatal("the new passphrase reads as saved")
	}
}

func TestReplaceKeepsThemWithDeleteOlderOff(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	first := w.taken(archive.Manual)
	replaced := w.replace(false)
	if got, want := w.listed(), []string{replaced.Name.String() + " verified", first.Name.String() + " verified"}; !slices.Equal(got, want) {
		t.Fatalf("listed %q, want %q", got, want)
	}
	if got, want := w.generations(), []backups.Generation{replaced.Generation, first.Generation}; !slices.Equal(got, want) {
		t.Fatalf("backup.key keeps generations %v, want %v", got, want)
	}
}

func TestDeletingTheNewestVerifiedBackupIsRefusedAndAnOlderOneGoes(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	older := w.taken(archive.Manual)
	newest := w.taken(archive.Manual)
	if err := w.backups.Delete(newest.ID); !errors.Is(err, backups.ErrNewestVerified) {
		t.Fatalf("deleting the newest good Backup = %v, want ErrNewestVerified", err)
	}
	if err := w.backups.Delete(older.ID); err != nil {
		t.Fatal(err)
	}
	if got, want := w.files(), []string{newest.Name.String()}; !slices.Equal(got, want) {
		t.Fatalf("the Backups folder holds %q, want %q", got, want)
	}
}

func (w *world) health() health.Answer {
	w.t.Helper()
	a, err := w.backups.HealthSource().Answer(w.t.Context())
	if err != nil {
		w.t.Fatal(err)
	}
	return a
}

// controllerItem is the Controller's Backups item, false while it has none.
func controllerItem(a health.Answer) (health.Item, bool) {
	for _, it := range a.Items {
		if it.Key == health.KeyOf("controller") {
			return it, true
		}
	}
	return health.Item{}, false
}

// failNext makes the next Backup fail for room.
func (w *world) failNext() {
	w.t.Helper()
	w.free.Store(0)
	if _, err := w.take(backups.Request{Trigger: archive.Manual}); err == nil {
		w.t.Fatal("a Backup with no room passed")
	}
	w.free.Store(1 << 40)
}

func TestHealthWarnsThenErrs(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	good := w.taken(archive.Scheduled)
	if _, ok := controllerItem(w.health()); ok {
		t.Fatal("a good Backup has a Health item")
	}
	w.failNext()
	a := w.health()
	item, ok := controllerItem(a)
	if !ok || item.Level != health.Warn || item.Subject != "Last Backup failed" || !strings.Contains(item.Text, "has 0 bytes free and the Backup needs about") {
		t.Fatalf("after a failed Backup the item is %+v, want a Warning naming the room", item)
	}
	if due := good.TakenAt.Add(2 * 24 * time.Hour); !a.Next.Equal(due) {
		t.Fatalf("Health asks again at %s, want %s", a.Next, due)
	}
	w.ahead.Store(int64(2*24*time.Hour + time.Minute))
	item, _ = controllerItem(w.health())
	if want := "No good Backup since " + good.TakenAt.UTC().Format("2 Jan"); item.Level != health.Fail || item.Subject != want {
		t.Fatalf("past Warn after times Back up every the item is %+v, want the Error %q", item, want)
	}
	w.taken(archive.Manual)
	if _, ok := controllerItem(w.health()); ok {
		t.Fatal("a good Backup after the failure left the item open")
	}
}

func TestHealthCountsOnlyLocalOrCopiedBackups(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	good := w.taken(archive.Scheduled)
	w.failNext()
	w.rewrite(good.ID, func(e *backups.Entry) { e.TakenAt = e.TakenAt.Add(-24 * time.Hour) })
	w.ahead.Store(int64(36 * time.Hour))
	levels := map[string]health.Level{}
	for name, edit := range map[string]func(*backups.Entry){
		"local":        func(e *backups.Entry) { e.Local, e.Copies = true, nil },
		"copied":       func(e *backups.Entry) { e.Local, e.Copies = false, []backups.Copy{{State: backups.Copied}} },
		"copy waiting": func(e *backups.Entry) { e.Local, e.Copies = false, []backups.Copy{{State: backups.CopyWaiting}} },
		"neither":      func(e *backups.Entry) { e.Local, e.Copies = false, nil },
	} {
		w.rewrite(good.ID, edit)
		w.opened("2.1.0")
		item, _ := controllerItem(w.health())
		levels[name] = item.Level
	}
	want := map[string]health.Level{"local": health.Fail, "copied": health.Fail, "copy waiting": health.Warn, "neither": health.Warn}
	for name, level := range want {
		if levels[name] != level {
			t.Errorf("with the good Backup %s the item is at %v, want %v: only a local file or a copy counts", name, levels[name], level)
		}
	}
}
