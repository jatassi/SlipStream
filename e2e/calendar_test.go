package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type calendarAnswer struct {
	Today string `json:"today"`
	Days  []struct {
		Day  string `json:"day"`
		Rows []struct {
			Work struct {
				Label string `json:"label"`
			} `json:"work"`
			Kind       string  `json:"kind"`
			ItemsWords string  `json:"itemsWords"`
			State      *string `json:"state"`
			Met        int     `json:"met"`
			Of         int     `json:"of"`
		} `json:"rows"`
	} `json:"days"`
}

// lines is each row as "day label: words kind state met/of".
func (c calendarAnswer) lines() []string {
	var out []string
	for _, d := range c.Days {
		for _, r := range d.Rows {
			state := "none"
			if r.State != nil {
				state = *r.State
			}
			out = append(out, fmt.Sprintf("%s %s: %s %s %s %d/%d", d.Day, r.Work.Label, r.ItemsWords, r.Kind, state, r.Met, r.Of))
		}
	}
	return out
}

// TestTheCalendarShowsThisMonthsAlbumMissingThenMetAfterItsImport dates
// Harbor Lights on the first day of the run's month and Night Ferry only
// by that month, adds The Lanterns with every track monitored, and reads
// the month: Harbor Lights is one row of three missing tracks and Night
// Ferry stays off. Its tracks then land in the music root, and after Scan
// now the row reads met.
func TestTheCalendarShowsThisMonthsAlbumMissingThenMetAfterItsImport(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "music-calendar"
	now := time.Now()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	next := first.AddDate(0, 1, 0)
	albums := s.Music.Artists[0].Revisions[0].Albums
	albums[0].FirstRelease = first.Format(time.DateOnly)
	albums[1].FirstRelease = first.Format("2006-01")
	track := func(name string) scenario.MediaFile {
		return scenario.MediaFile{
			Path:  "The Lanterns/Harbor Lights/" + name + " 1080p BluRay.mkv",
			Video: scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 1, HDR: "sdr"},
			Audio: []scenario.MediaAudio{{Codec: "aac", Channels: 2, Language: "eng"}},
		}
	}
	s.Library = &scenario.Library{
		Roots:   []scenario.LibraryRoot{{ContentType: "music", Folder: "music"}},
		Folders: []scenario.LibraryFolder{{Folder: "incoming", Files: []scenario.MediaFile{track("01 Tidewater"), track("02 Paper Boats"), track("03 Low Bridge")}}},
	}
	run := worldtest.LaunchE2EScenario(t, s)
	a := api(t, run)
	decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"music","key":"`+lanterns+`","preset":"all"}`), &added{})

	month := "/api/v1/calendar?from=" + first.Format(time.DateOnly) + "&to=" + next.Format(time.DateOnly)
	read := func() []string {
		var c calendarAnswer
		return decode(t, "getCalendar", a.Admin("home", http.MethodGet, month, ""), &c).lines()
	}
	day := first.Format(time.DateOnly)
	missing := []string{day + " The Lanterns: Harbor Lights · Tracks 1 to 3 release missing 0/3"}
	if got := read(); fmt.Sprint(got) != fmt.Sprint(missing) {
		t.Fatalf("this month reads %q, want %q", got, missing)
	}

	var roots struct {
		Roots []struct {
			ID   string `json:"id"`
			Scan struct {
				LastAt  *time.Time `json:"lastAt"`
				Running *struct{}  `json:"running"`
			} `json:"scan"`
		} `json:"roots"`
	}
	worldtest.Until(t, "the music root's first library scan finished", func() (bool, error) {
		decode(t, "listRoots", a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &roots)
		return len(roots.Roots) == 1 && roots.Roots[0].Scan.LastAt != nil && roots.Roots[0].Scan.Running == nil, nil
	})
	media := run.Manifest().MediaDir
	if err := os.Rename(filepath.Join(media, "incoming", "The Lanterns"), filepath.Join(media, "music", "The Lanterns")); err != nil {
		t.Fatal(err)
	}
	if r := a.Admin("home", http.MethodPost, "/api/v1/roots/"+roots.Roots[0].ID+"/scan", ""); r.Status != http.StatusAccepted {
		t.Fatalf("scanRoot = %d %s, want 202", r.Status, r.Body)
	}
	met := []string{day + " The Lanterns: Harbor Lights · Tracks 1 to 3 release met 3/3"}
	worldtest.Until(t, "Harbor Lights reads met once its tracks are imported", func() (bool, error) {
		return fmt.Sprint(read()) == fmt.Sprint(met), nil
	})
}
