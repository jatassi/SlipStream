package sim_test

import (
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// misdescribed is the packs world's misdescribed release named for the
// 1080p Target, which its SDR video meets: its title says HDR10.
const misdescribed = "The.Glass.Meridian.2020.1080p.WEB-DL.HDR10.H265-MIS"

// meridian launches the packs world offering only release, with samples
// under sample percent of the runtime, and adds The Glass Meridian, whose
// search grabs it.
func meridian(t *testing.T, release string, sample int) *sim.World {
	return grabOnly(t, "9100001", release, sample)
}

// grabOnly launches the packs world offering only release, with samples
// under sample percent of the runtime, and adds the movie key, whose
// search grabs it.
func grabOnly(t *testing.T, key, release string, sample int) *sim.World {
	t.Helper()
	s, err := scenario.Load("packs")
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range s.Releases {
		if r.Name == "The.Glass.Meridian.2020.2160p.WEB-DL.H265-MIS" {
			s.Releases[i].Name, s.Releases[i].Files[0].Path = misdescribed, misdescribed+".mkv"
		}
	}
	s.Releases = slices.DeleteFunc(s.Releases, func(r scenario.Release) bool { return r.Name != release })
	s.Library.Importing.SampleShorterThan = sample
	w := sim.Launch(t, s, sim.WithWalk())
	added := simtest.Decode[struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","preset":"all"}`))
	simtest.Search(t, w, added.Work.ID)
	return w
}

// blocklisted is each Blocklist entry's title and cause.
func blocklisted(t *testing.T, w *sim.World) []string {
	t.Helper()
	var out []string
	for _, e := range simtest.Decode[struct {
		Entries []struct {
			Title string `json:"title"`
			Cause string `json:"cause"`
		} `json:"entries"`
	}](t, "listBlocklist", w.API().Admin("home", http.MethodGet, "/api/v1/blocklist", "")).Entries {
		out = append(out, e.Title+" "+e.Cause)
	}
	return out
}

// TestEachReleaseFaultFailsOrWaitsWithItsBlocklistCause grabs one release
// of each row of the release-fault table: a download with no video file,
// or none past the name and folder rules, fails no_video_file and is
// blocklisted no_video; one ffprobe can't read twice fails unreadable and
// is blocklisted unreadable; one whose probe contradicts its title on
// every video imports and is blocklisted misdescribed; one whose videos
// only the sample setting leaves out waits settings and isn't blocklisted.
func TestEachReleaseFaultFailsOrWaitsWithItsBlocklistCause(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		release string
		sample  int
		stage   string
		failure string
		cause   string
	}{
		{"The.Glass.Meridian.2020.720p.WEB-DL.H264-SMP", 5, "failed", "no_video_file", "no_video"},
		{"The.Glass.Meridian.2020.1080p.BluRay.x264-RAR", 5, "failed", "no_video_file", "no_video"},
		{"The.Glass.Meridian.2020.1080p.WEB-DL.H264-BAD", 5, "failed", "unreadable", "unreadable"},
		{misdescribed, 5, "imported", "", "misdescribed"},
	} {
		t.Run(c.release, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				w := meridian(t, c.release, c.sample)
				g := simtest.Await(t, w)
				var code string
				if g.Failure != nil {
					code = g.Failure.Code
				}
				if g.Stage != c.stage || code != c.failure {
					t.Fatalf("the Grab of %s ended %s %q, want %s %q", c.release, g.Stage, code, c.stage, c.failure)
				}
				if got, want := blocklisted(t, w), []string{c.release + " " + c.cause}; !slices.Equal(got, want) {
					t.Fatalf("the Blocklist holds %q, want %q", got, want)
				}
			})
		})
	}
}

// TestVideosOnlyTheSampleSettingLeavesOutWait grabs The Glass Meridian at
// 1080p and 2160p with samples under half the runtime: both videos are
// samples by the setting alone, so they wait settings and nothing is
// blocklisted.
func TestVideosOnlyTheSampleSettingLeavesOutWait(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := meridian(t, "The.Glass.Meridian.2020.1080p.2160p.BluRay.x265-FKW", 50)
		var v []importView
		for range simtest.Settled / time.Second {
			if v = grabImports(t, w); len(v) == 1 && v[0].count("waiting") == 2 {
				break
			}
			time.Sleep(time.Second)
		}
		if len(v) != 1 || !slices.Equal(v[0].outcomes(), []string{
			"The.Glass.Meridian.2020.1080p.BluRay.x264-FKW.mkv: waiting settings",
			"The.Glass.Meridian.2020.2160p.BluRay.HDR10.x265-FKW.mkv: waiting settings",
		}) {
			t.Fatalf("the Grab's Imports read %+v, want both videos waiting settings", v)
		}
		if got := blocklisted(t, w); len(got) != 0 {
			t.Fatalf("the Blocklist holds %q, want nothing", got)
		}
	})
}

// TestNotThisReleaseBlocklistsAWaitingDownload grabs a release named for
// 1080p whose one video is a 480p file, which the 1080p Target doesn't
// take: the
// probe contradicts the title, so it is blocklisted misdescribed, and the
// video waits meets_nothing. Not this Release on it declines it and fails
// the Grab blocklisted, and the Blocklist keeps the first entry.
func TestNotThisReleaseBlocklistsAWaitingDownload(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		const release, video = "The.Glass.Meridian.2020.1080p.WEB-DL.H265-MIS", "The.Glass.Meridian.2020.480p.WEB-DL.H265-MIS.mkv"
		s, err := scenario.Load("packs")
		if err != nil {
			t.Fatal(err)
		}
		s.Releases = slices.DeleteFunc(s.Releases, func(r scenario.Release) bool { return r.Name != "The.Glass.Meridian.2020.2160p.WEB-DL.H265-MIS" })
		s.Releases[0].Name, s.Releases[0].Files[0].Path = release, video
		s.Releases[0].Files[0].Video = &scenario.MediaVideo{Width: 640, Height: 480, Seconds: 8, HDR: "sdr"}
		w := sim.Launch(t, s, sim.WithWalk())
		simtest.Search(t, w, simtest.Decode[struct {
			Work struct {
				ID string `json:"id"`
			} `json:"work"`
		}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9100001","preset":"all"}`)).Work.ID)
		var v []importView
		for range simtest.Settled / time.Second {
			if v = grabImports(t, w); len(v) == 1 && v[0].count("waiting") == 1 {
				break
			}
			time.Sleep(time.Second)
		}
		if len(v) != 1 || !slices.Equal(v[0].outcomes(), []string{video + ": waiting meets_nothing"}) {
			t.Fatalf("the Grab's Imports read %+v, want its video waiting meets_nothing; the Grabs read %+v", v, simtest.GrabsListed(t, w))
		}
		if r := w.API().Admin("home", http.MethodPost, "/api/v1/imports/"+v[0].ID+"/files/0", `{"action":"notThisRelease"}`); r.Status != http.StatusOK {
			t.Fatalf("actOnImportFile notThisRelease = %d %s, want 200", r.Status, r.Body)
		}
		g := simtest.Await(t, w)
		if g.Stage != "failed" || g.Failure == nil || g.Failure.Code != "blocklisted" {
			t.Fatalf("the Grab ended %s with %+v, want failed blocklisted", g.Stage, g.Failure)
		}
		if got, want := blocklisted(t, w), []string{release + " misdescribed"}; !slices.Equal(got, want) {
			t.Fatalf("the Blocklist holds %q, want %q", got, want)
		}
	})
}
