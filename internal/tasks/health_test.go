package tasks_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

func failing(id tasks.TaskID, name, category string, broken *atomic.Bool) tasks.Task {
	t := tidy(nil)
	t.ID, t.Name, t.Health = id, name, category
	t.Run = func(context.Context, tasks.Unit, *tasks.Progress) error {
		if broken.Load() {
			return errors.New("database is locked")
		}
		return nil
	}
	return t
}

func answer(t *testing.T, s *tasks.Scheduler, category health.ID) []string {
	t.Helper()
	a, err := s.HealthSource(category).Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, it := range a.Items {
		var facts []string
		for _, f := range it.Detail {
			facts = append(facts, f.Label)
		}
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%s %s|%s", it.Key, it.Level, it.Subject, it.Text, it.Action.Kind(), it.Action.Task(), strings.Join(facts, ",")))
	}
	for _, u := range a.Unsure {
		out = append(out, fmt.Sprintf("unsure %s|%s|%s", u.Key, u.Subject, u.Text))
	}
	return out
}

func TestHealthSourceRaisesAFailedTaskInItsCategory(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		var broken atomic.Bool
		broken.Store(true)
		s := open(t, lib, failing("tidy", "Tidy", "", &broken), failing("backup", "Backup", "backups", &broken))
		clock(t, s)
		for _, id := range []tasks.TaskID{"tidy", "backup"} {
			if _, err := s.RunNow(t.Context(), id, admin); err != nil {
				t.Fatal(err)
			}
			synctest.Wait()
		}
		failedAt := "00:00"
		want := []string{"tidy|warn|Tidy|Tidy failed at " + failedAt + ": database is locked. Run it again once the cause is fixed.|run tidy|Failed at,Cause"}
		if got := answer(t, s, health.ScheduledTasksID); !equal(got, want) {
			t.Fatalf("scheduled tasks = %q, want %q", got, want)
		}
		want = []string{"backup|warn|Backup|Backup failed at " + failedAt + ": database is locked. Run it again once the cause is fixed.|run backup|Failed at,Cause"}
		if got := answer(t, s, "backups"); !equal(got, want) {
			t.Fatalf("backups = %q, want %q", got, want)
		}

		broken.Store(false)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := answer(t, s, health.ScheduledTasksID); len(got) != 0 {
			t.Fatalf("scheduled tasks after a success = %q, want none", got)
		}
	})
}

func TestHealthSourceIsUnsureOfATaskWaitingForItsNode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		var broken atomic.Bool
		broken.Store(true)
		far := failing("tidy", "Tidy", "", &broken)
		far.Units = func(context.Context, storage.Reader[storage.Library]) ([]tasks.Unit, error) {
			return []tasks.Unit{{Key: "s", Label: "S", Place: tasks.TiedTo(seedbox)}}, nil
		}
		r := &links{link: nodes.Up}
		s := openOn(t, lib, r, newHub(t), far)
		clock(t, s)
		if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		r.set(nodes.Down)
		want := []string{"unsure tidy|Tidy|Tidy is waiting for seedbox and runs again once it is back."}
		if got := answer(t, s, health.ScheduledTasksID); !equal(got, want) {
			t.Fatalf("scheduled tasks = %q, want %q", got, want)
		}
	})
}
