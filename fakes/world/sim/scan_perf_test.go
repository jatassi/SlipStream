// The race detector slows the walk several times over, so its timing says
// nothing about the 60 s target and it alone ran 250 s of the package's
// 20 minute budget. The unchanged rescan is proved under -race at small
// scale by TestAFirstScanAddsTheWorksItIsSureOfAndAnUnchangedRootReadsNothing.

//go:build !race

package sim_test

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// TestAScanOfFiftyThousandUnchangedFilesReadsNone shelves 50,000 files in
// 50 Work folders on a local disk: once the first scan records them, the
// next reads none of them, and the test logs how long that scan took on
// the real clock.
func TestAScanOfFiftyThousandUnchangedFilesReadsNone(t *testing.T) {
	if testing.Short() {
		t.Skip("walks 50,000 files on disk")
	}
	const works, each = 50, 1000
	shelved := make([]scenario.TMDBMovie, works)
	for i := range shelved {
		shelved[i] = scenario.TMDBMovie{ID: int64(9200000 + i), IMDbID: fmt.Sprintf("tt92%05d", i), Title: fmt.Sprintf("Long Shelf %02d", i), Released: "2016-01-01", Runtime: 90 * time.Minute}
	}
	clock := wallClock(t)
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, func(sc *scenario.Scenario) { sc.TMDB.Movies = append(sc.TMDB.Movies, shelved...) })
		for _, m := range shelved {
			for j := range each {
				s.put(fmt.Sprintf("%s/%s part %04d.mkv", folderOf(m), m.Title, j), 1, hd)
			}
		}
		expectScanned(t, s.scan(), "succeeded", succeeded)
		if got := len(s.probes.took()); got != works*each {
			t.Fatalf("the first scan read %d files, want %d", got, works*each)
		}

		began := clock()
		expectScanned(t, s.scan(), "succeeded", succeeded)
		took := clock().Sub(began)
		s.expectReads("a scan of 50,000 unchanged files")
		t.Logf("a scan of %d unchanged files took %s", works*each, took)
	})
}

// wallClock reads the real clock from inside a synctest bubble, whose own
// clock is fake.
func wallClock(t *testing.T) func() time.Time {
	ask, answer := make(chan struct{}), make(chan time.Time)
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	go func() {
		for {
			select {
			case <-ask:
				answer <- time.Now()
			case <-done:
				return
			}
		}
	}()
	return func() time.Time {
		ask <- struct{}{}
		return <-answer
	}
}
