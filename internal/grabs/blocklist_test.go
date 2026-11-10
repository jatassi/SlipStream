package grabs_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	bluRay = "The.Attic.2020.1080p.BluRay.x264-GRP"
	webDL  = "The.Attic.2020.1080p.WEB-DL-GRP"
)

func hashed(letter string) bencode.InfoHash {
	return *listing("", 0, letter).InfoHash
}

func entry(h bencode.InfoHash, title string) grabs.BlockEntry {
	return grabs.BlockEntry{InfoHash: h, Title: title, Size: 4096, Cause: grabs.CauseByHand, By: &grabs.Actor{Name: "Jackson"}}
}

func reconciling(yes bool) grabs.BlocklistOn {
	return grabs.BlocklistOn{Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return yes, nil }}
}

func (tw twinWorld) page(t *testing.T, q grabs.BlocklistQuery) grabs.BlocklistPage {
	t.Helper()
	page, err := tw.g.Blocklist(t.Context(), identity.Admin{}, q)
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func TestMarkFailedBlocklistsTheReleaseOfADownloadingGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		downloadingGrab(t, p, stalledReport)
		torrent := torrentOf(t, p)
		run(t, p.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling(false).MarkedFailed(ctx, tx, []downloadclients.ID{torrent}, identity.Admin{})
		})

		rel := newRelease(t)
		again := p.grab(rel.link.String(), nil)
		p.running()
		p.failed(again, grabs.Blocklisted)
	})
}

func TestTheBlocklistKeepsTheFirstEntryOfAHashAndRemovesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tw := newTwinWorld(t, twin{})
		a, b := hashed("a"), hashed("b")
		first := entry(a, bluRay)
		first.Work = tw.work
		grabstest.Blocked(t, tw.lib, first)
		time.Sleep(time.Second)
		second := entry(a, webDL)
		second.Cause, second.By = grabs.CauseMarkedFailed, &grabs.Actor{}
		grabstest.Blocked(t, tw.lib, second)
		grabstest.Blocked(t, tw.lib, entry(b, webDL))

		page := tw.page(t, grabs.BlocklistQuery{})
		if page.Total != 2 || len(page.Entries) != 2 {
			t.Fatalf("the Blocklist holds %+v, want two entries", page)
		}
		got := page.Entries[1]
		if got.InfoHash != a || got.Title != bluRay || got.Because != "Jackson blocklisted it when canceling its Grab." ||
			got.WorkTitle != "The Attic" || got.Words != "Bluray-1080p · GRP" {
			t.Fatalf("a's entry is %+v, want the first one added", got)
		}

		for range 2 {
			if err := tw.g.Unblock(t.Context(), identity.Admin{}, []bencode.InfoHash{a}); err != nil {
				t.Fatal(err)
			}
		}
		if page := tw.page(t, grabs.BlocklistQuery{}); page.Total != 1 || page.Entries[0].InfoHash != b {
			t.Fatalf("the Blocklist holds %+v after removing a twice, want b's entry alone", page)
		}
	})
}

func TestTheBlocklistListsPagesOfFiftyNewestFirstFilteredByTitle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tw := newTwinWorld(t, twin{})
		for i := range 51 {
			title := fmt.Sprintf("The.Attic.2020.Part%02d.1080p.WEB-DL", i)
			if i%10 == 0 {
				title = fmt.Sprintf("The.Cellar.2020.Part%02d.1080p.WEB-DL", i)
			}
			grabstest.Blocked(t, tw.lib, entry(bencode.InfoHash([]byte(fmt.Sprintf("%020d", i))), title))
			time.Sleep(time.Second)
		}

		first, rest := tw.page(t, grabs.BlocklistQuery{}), tw.page(t, grabs.BlocklistQuery{Offset: 50})
		if first.Total != 51 || len(first.Entries) != 50 || first.Entries[0].Title != "The.Cellar.2020.Part50.1080p.WEB-DL" ||
			len(rest.Entries) != 1 || rest.Entries[0].Title != "The.Cellar.2020.Part00.1080p.WEB-DL" {
			t.Fatalf("the pages hold %d of %d from %q, then %+v; want 50 of 51 from the newest, then the oldest", len(first.Entries), first.Total, first.Entries[0].Title, rest.Entries)
		}
		cellar := tw.page(t, grabs.BlocklistQuery{Title: "cellar"})
		var titles []string
		for _, e := range cellar.Entries {
			titles = append(titles, e.Title)
		}
		want := []string{
			"The.Cellar.2020.Part50.1080p.WEB-DL", "The.Cellar.2020.Part40.1080p.WEB-DL", "The.Cellar.2020.Part30.1080p.WEB-DL",
			"The.Cellar.2020.Part20.1080p.WEB-DL", "The.Cellar.2020.Part10.1080p.WEB-DL", "The.Cellar.2020.Part00.1080p.WEB-DL",
		}
		if cellar.Total != 6 || !slices.Equal(titles, want) {
			t.Fatalf("q=cellar lists %q of %d, want %q", titles, cellar.Total, want)
		}
	})
}

func TestNothingIsBlocklistedWhileReconciling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tw := newTwinWorld(t, twin{})
		err := tw.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling(true).Block(ctx, tx, entry(hashed("a"), bluRay))
		})
		if !errors.Is(err, grabs.ErrReconciling) {
			t.Fatalf("Block while Reconciling = %v, want ErrReconciling", err)
		}
		if page := tw.page(t, grabs.BlocklistQuery{}); page.Total != 0 {
			t.Fatalf("the Blocklist holds %+v, want nothing", page)
		}
	})
}
