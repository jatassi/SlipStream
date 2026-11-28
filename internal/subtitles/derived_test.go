package subtitles_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const filmStem = "Film/Film.2024.1080p.WEB-DL-GRP"

// downloadedSource is a film wanting each of langs whose source file has
// SubDL's English and French and a found Dutch beside it.
func downloadedSource(t *testing.T, w *subtitlestest.World, langs map[string]targets.WantedLanguage) (tree works.Tree, source files.ID) {
	t.Helper()
	tree = w.Work("film", w.Profile("Films", langs["en"], langs["fr"], langs["de"], langs["nl"]))
	w.Put(filmStem+".nl.srt", onSpeech)
	source = w.Place(tree, filmStem+".mkv", audio("ja"))
	w.Sweep()
	keyOn(t, w)
	for _, tag := range []string{"en", "fr"} {
		w.Catalog.List(subtitlestest.Listing{
			Offer: subtitling.Offer{Provider: "subdl", Key: tag, Language: langs[tag].Language, Releases: []string{"Film.2024.1080p.WEB-DL-GRP"}, Downloads: 1},
			Name:  tag + ".srt", Bytes: []byte(onSpeech + "\n2\n00:00:03,000 --> 00:00:03,500\n" + tag + "\n"),
		})
		searchNow(t, w, subtitles.WantKey{File: source, Want: subtitles.Want{Language: langs[tag].Language}})
	}
	if got := rows(t, w); !slices.Equal(got, []string{filmStem + ".mkv en provider", filmStem + ".mkv fr provider", filmStem + ".mkv nl found untimed"}) {
		t.Fatalf("before the Conversion the source's Sidecars are %v", got)
	}
	return tree, source
}

// land adds the made file at rel with m, then Landed and Derived, in one
// landing Write.
func land(t *testing.T, w *subtitlestest.World, tree works.Tree, source files.ID, rel string, m media.Record) files.ID {
	t.Helper()
	w.Put(rel, strings.Repeat("v", 1024))
	var made files.ID
	w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if made, err = w.Add(ctx, tx, tree, rel, m); err != nil {
			return err
		}
		if err := w.Subs.Landed(ctx, tx, files.File{ID: made}); err != nil {
			return err
		}
		return w.Subs.Derived(ctx, tx, source, made)
	})
	return made
}

func TestAMadeFileKeepsItsTracksCopiesTheDownloadedSidecarsAndSearchesTheRest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		langs := map[string]targets.WantedLanguage{}
		for _, tag := range []string{"en", "fr", "de", "nl"} {
			langs[tag] = subtitlestest.Wanted(t, tag, false, targets.Allow, targets.Always)
		}
		tree, source := downloadedSource(t, w, langs)
		withFrench := audio("ja")
		withFrench.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "fr"}}
		m := land(t, w, tree, source, filmStem+" - SDR.mkv", withFrench)
		w.Sweep()

		if st := outcome(t, w, m, subtitles.Want{Language: langs["fr"].Language}); st.Kind != subtitles.Embedded {
			t.Errorf("the made file's French reads %+v, want met by its MKV track", st)
		}
		if got := query(t, w, `SELECT language || ' ' || source FROM subtitles_sidecar WHERE file_id = ? ORDER BY language`, m); !slices.Equal(got, []string{"en sibling", "nl sibling"}) {
			t.Errorf("the made file's Sidecars are %v, want the source's English copied and its found Dutch timed by a search", got)
		}
		was, copied := w.On(filmStem+".en.srt"), w.On(filmStem+" - SDR.en.srt")
		if was == "" || copied != was {
			t.Errorf("the made file's English holds %q and the source's %q, want the same bytes in both", copied, was)
		}
		if got := query(t, w, `SELECT language FROM subtitles_want WHERE file_id = ? AND last_result IS NOT NULL ORDER BY language`, m); !slices.Equal(got, []string{"de", "nl"}) {
			t.Errorf("the made file searched %v, want only what neither it nor the source's downloads have", got)
		}
	})
}
