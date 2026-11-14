package storing

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/media"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/rpc"
)

func walked(t *testing.T, w walking, videos media.Videos) []string {
	t.Helper()
	p, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1000, videos)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range p.Entries {
		out = append(out, e.Path.String())
	}
	return out
}

func TestTheWalkListsAnExtensionOnlyWhenTheListHasIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat (1995)/Heat (1995).mkv", "film")
		w.file(t, "Orchard (2020)/Orchard (2020).WEBM", "film")
		if got, want := walked(t, w, media.NewVideos("mkv")), []string{"Heat (1995)/Heat (1995).mkv"}; !slices.Equal(got, want) {
			t.Errorf("a walk for mkv lists %v, want %v", got, want)
		}
		if got, want := walked(t, w, media.NewVideos(".MKV", "webm")), []string{"Heat (1995)/Heat (1995).mkv", "Orchard (2020)/Orchard (2020).WEBM"}; !slices.Equal(got, want) {
			t.Errorf("a walk for mkv and webm lists %v, want %v", got, want)
		}
	})
}

func TestCompanionsFollowTheVideoList(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, f := range []string{"Heat/Heat.mkv", "Heat/Heat.webm", "Heat/Heat.en.srt"} {
			w.file(t, f, f)
		}
		from, to := mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv")
		for _, c := range []struct {
			videos media.Videos
			want   []Rel
		}{
			{media.NewVideos("mkv", "webm"), rels(t, "Heat/Heat.en.srt")},
			{media.NewVideos("mkv"), rels(t, "Heat/Heat.en.srt", "Heat/Heat.webm")},
		} {
			surveyed, err := w.remote.Survey(t.Context(), w.at, []Planned{{From: from, To: to}}, true, c.videos)
			if err != nil {
				t.Fatal(err)
			}
			if got := surveyed[0].Companions; !slices.Equal(got, c.want) {
				t.Errorf("with videos %v, Heat.mkv's companions are %v, want %v", c.videos.Extensions(), got, c.want)
			}
		}
		if _, err := w.remote.Move(t.Context(), w.at, from, to, true, media.NewVideos("mkv", "webm")); err != nil {
			t.Fatal(err)
		}
		want := []string{"Heat (1995)/Heat (1995).en.srt", "Heat (1995)/Heat (1995).mkv", "Heat/Heat.webm"}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Errorf("after the move the root holds %v, want %v", got, want)
		}
	})
}

func TestARequestWithNoVideoExtensionsIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		root := w.at.Root.Bytes()
		for name, call := range map[string]func() error{
			"Walk": func() error {
				_, err := w.store.Walk(t.Context(), &storagev1.WalkRequest{RootId: root, Limit: 1})
				return err
			},
			"Move": func() error {
				_, err := w.store.Move(t.Context(), &storagev1.MoveRequest{RootId: root, From: "a.mkv", To: "b.mkv"})
				return err
			},
			"Survey": func() error {
				_, err := w.store.Survey(t.Context(), &storagev1.SurveyRequest{RootId: root})
				return err
			},
		} {
			if err := call(); rpc.CodeOf(err) != rpc.CodeInvalidArgument {
				t.Errorf("%s with no video extensions = %v, want InvalidArgument", name, err)
			}
		}
	})
}
