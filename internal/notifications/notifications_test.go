package notifications

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/internal/db"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type grabbedFacts struct {
	Release string `json:"release"`
}

var (
	grabbedRow = notice.Row[grabbedFacts]{Spec: notice.Spec{Kind: "grabbed", Label: "Grabbed", Group: notice.Grabs}}
	failedRow  = notice.Row[grabbedFacts]{Spec: notice.Spec{Kind: "grab_failed", Label: "Grab failed", Group: notice.Grabs}}
	deletedRow = notice.Row[grabbedFacts]{Spec: notice.Spec{Kind: "works_deleted", Label: "Works deleted", Group: notice.Library}}
)

func TestARecordIsAnUnreleasedEventOnlyWhenItsWriteCommits(t *testing.T) {
	ctx := t.Context()
	lib := storagetest.Library(t, Schema())
	s := New(Config{Library: lib})
	rec := s.Recorder("grabs", []notice.Entry{grabbedRow})

	failed := errors.New("the Grab's Write failed")
	err := lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := grabbedRow.Record(ctx, tx, rec, grabbedFacts{Release: "Heat.1995.1080p"}); err != nil {
			return err
		}
		return failed
	})
	if !errors.Is(err, failed) {
		t.Fatal(err)
	}
	if got := unreleased(t, lib); len(got) != 0 {
		t.Fatalf("a rolled-back Write left %d events", len(got))
	}
	<-s.doorbell

	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return grabbedRow.Record(ctx, tx, rec, grabbedFacts{Release: "Heat.1995.2160p"})
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.doorbell:
	default:
		t.Fatal("the doorbell didn't ring")
	}
	got := unreleased(t, lib)
	if len(got) != 1 || got[0].Kind != grabbedRow.Kind || got[0].Subject != `{"release":"Heat.1995.2160p"}` || got[0].DueAt != got[0].At {
		t.Fatalf("unreleased = %+v, want one grabbed event due when it happened", got)
	}
}

func unreleased(t *testing.T, lib *storage.DB[storage.Library]) []db.NotificationsEvent {
	t.Helper()
	got, err := db.New(lib).ListDueUnreleased(t.Context(), db.ListDueUnreleasedParams{Now: storage.Now(), Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestTheCatalogListsHealthThenOwnersThenTest(t *testing.T) {
	s := New(Config{})
	s.Recorder("grabs", []notice.Entry{grabbedRow, failedRow})
	s.Recorder("library", []notice.Entry{deletedRow})
	var got []notice.Kind
	for _, spec := range s.Events() {
		got = append(got, spec.Kind)
	}
	if want := []notice.Kind{"new_health_item", "health_item_cleared", "grabbed", "grab_failed", "works_deleted", "test"}; !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestWiringBugsPanic(t *testing.T) {
	panics := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s didn't panic", name)
			}
		}()
		f()
	}
	panics("a kind two owners register", func() {
		s := New(Config{})
		s.Recorder("grabs", []notice.Entry{grabbedRow})
		s.Recorder("imports", []notice.Entry{grabbedRow})
	})
	panics("an owner registering a Health kind", func() {
		New(Config{}).Recorder("grabs", []notice.Entry{notice.Row[grabbedFacts]{Spec: notice.Spec{Kind: newItemsRow.Kind}}})
	})
	panics("an owner registering test", func() {
		New(Config{}).Recorder("grabs", []notice.Entry{notice.Row[grabbedFacts]{Spec: notice.Spec{Kind: testRow.Kind}}})
	})
	panics("registering after the dispatcher started", func() {
		s := New(Config{})
		s.catalog.seal()
		s.Recorder("grabs", []notice.Entry{grabbedRow})
	})
	panics("two providers with one id", func() {
		New(Config{Providers: []providers.Provider{{ID: "discord"}, {ID: "slack"}, {ID: "discord"}}})
	})
}
