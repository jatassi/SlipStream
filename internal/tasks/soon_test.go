package tasks_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
)

func released() chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

func TestSoonWithNoOpenRunRunsNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		s := open(t, lib, tidy(released()))
		clock(t, s)
		time.Sleep(time.Hour)
		synctest.Wait()
		asked := time.Now().UnixMicro()
		s.Soon("tidy")
		synctest.Wait()
		want := []string{fmt.Sprintf("schedule|succeeded|%d", asked)}
		if got := rows(t, lib, "SELECT kind, result, started_at FROM tasks_run ORDER BY id"); !equal(got, want) {
			t.Fatalf("runs = %q, want %q", got, want)
		}
	})
}

func TestSoonDuringARunRunsOnceAfterIt(t *testing.T) {
	for _, calls := range []int{1, 3} {
		t.Run(fmt.Sprintf("%d calls", calls), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				lib := storagetest.Library(t, tasks.Schema())
				release := make(chan struct{})
				s := open(t, lib, tidy(release))
				clock(t, s)
				if _, err := s.RunNow(t.Context(), "tidy", admin); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				for range calls {
					s.Soon("tidy")
					time.Sleep(time.Minute)
					synctest.Wait()
				}
				if got := rows(t, lib, "SELECT kind, result FROM tasks_run ORDER BY id"); !equal(got, []string{"run-now|NULL"}) {
					t.Fatalf("runs while the first is open = %q, want only it", got)
				}
				close(release)
				time.Sleep(time.Hour)
				synctest.Wait()
				want := []string{"run-now|succeeded", "schedule|succeeded"}
				if got := rows(t, lib, "SELECT kind, result FROM tasks_run ORDER BY id"); !equal(got, want) {
					t.Fatalf("runs = %q, want %q", got, want)
				}
			})
		})
	}
}

func TestSoonMovesTheRegularSlot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, tasks.Schema())
		s := open(t, lib, tidy(released()))
		start := time.Now()
		clock(t, s)
		time.Sleep(time.Hour)
		synctest.Wait()
		s.Soon("tidy")
		synctest.Wait()
		time.Sleep(7*time.Hour + 10*time.Minute)
		synctest.Wait()
		if got := rows(t, lib, "SELECT kind FROM tasks_run ORDER BY id"); !equal(got, []string{"schedule"}) {
			t.Fatalf("runs just past the slot Soon replaced = %q, want only the Soon run", got)
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		got := rows(t, lib, fmt.Sprintf("SELECT kind, (started_at - %d) / 60000000 FROM tasks_run ORDER BY id", start.UnixMicro()))
		if len(got) != 2 || got[0] != "schedule|60" {
			t.Fatalf("runs = %q, want the Soon run an hour in, then one more a cadence later", got)
		}
		var minutes int
		if _, err := fmt.Sscanf(got[1], "schedule|%d", &minutes); err != nil || minutes < 9*60 || minutes > 9*60+5 {
			t.Fatalf("the next scheduled run started %q minutes in, want 8 h after the Soon run plus at most 5 minutes of jitter", got[1])
		}
	})
}
