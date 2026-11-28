package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const (
	grandma   = "/media/shelf/Grandma's 80th Birthday (2019).mkv"
	orchardTo = "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 720p mobile.mkv"
)

type conversionRow struct {
	Stage string `json:"stage"`
	Work  struct {
		ID string `json:"id"`
	} `json:"work"`
	Wait *struct {
		Code    string   `json:"code"`
		Servers []string `json:"servers"`
	} `json:"wait"`
}

type conversionList struct {
	Converting []conversionRow `json:"converting"`
	Next       []struct {
		Rows []json.RawMessage `json:"rows"`
	} `json:"next"`
}

func (l conversionList) rows(t *testing.T) []conversionRow {
	t.Helper()
	out := slices.Clone(l.Converting)
	for _, lane := range l.Next {
		for _, raw := range lane.Rows {
			var r conversionRow
			if err := json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
			out = append(out, r)
		}
	}
	return out
}

type madeFile struct {
	Path     string `json:"path"`
	PlacedBy string `json:"placedBy"`
	MadeFrom *struct {
		Name string `json:"name"`
	} `json:"madeFrom"`
	Media struct {
		Video struct {
			Codec  string `json:"codec"`
			Height int    `json:"height"`
		} `json:"video"`
	} `json:"media"`
}

// TestAConversionWaitsForAPlexTranscodeThenLands launches the conversion
// Scenario with Living room Plex transcoding video on home, so The Quiet
// Orchard's Conversion to 720p mobile is placed and waits, and once the
// play stops the release binary converts the 1080p file on the processor
// and the made file lands in the Work's folder.
func TestAConversionWaitsForAPlexTranscodeThenLands(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "conversion")
	m, a := run.Manifest(), api(t, run)
	played, err := run.Cue(worldtest.Context(t), "plex-play", json.RawMessage(`{"path":"`+grandma+`","user":"plex-admin","player":"Office PC","playerKey":"office-pc",`+
		`"video":"transcode","audio":"direct","subtitles":"none","received":{"container":"mkv","videoCodec":"h264","height":720,"audioCodec":"aac","channels":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	var session struct {
		Result struct {
			Session string `json:"session"`
		} `json:"result"`
	}
	if err := json.Unmarshal(played.Data, &session); err != nil {
		t.Fatal(err)
	}

	first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","preset":"all"}`), &added{})
	var work tree
	decodeRaw(t, first.Work, &work)
	a.Searched(work.ID, "", 1)

	var waiting conversionRow
	worldtest.Until(t, "the Conversion is placed and waits for the transcode", func() (bool, error) {
		var l conversionList
		decode(t, "listConversions", a.Admin("home", http.MethodGet, "/api/v1/conversions", ""), &l)
		i := slices.IndexFunc(l.rows(t), func(r conversionRow) bool { return r.Work.ID == work.ID })
		if i < 0 {
			return false, nil
		}
		waiting = l.rows(t)[i]
		return waiting.Stage == "ready", nil
	})
	if waiting.Wait == nil || waiting.Wait.Code != "transcode" || !slices.Equal(waiting.Wait.Servers, []string{"Living room Plex"}) {
		t.Fatalf("the placed Conversion waits %+v, want transcode on Living room Plex", waiting.Wait)
	}

	if _, err := run.Cue(worldtest.Context(t), "plex-stop", json.RawMessage(`{"session":"`+session.Result.Session+`"}`)); err != nil {
		t.Fatal(err)
	}
	var made madeFile
	worldtest.Until(t, "the made file lands", func() (bool, error) {
		var l struct {
			Files []madeFile `json:"files"`
		}
		decode(t, "listWorkFiles", a.Admin("home", http.MethodGet, "/api/v1/works/"+work.ID+"/files", ""), &l)
		i := slices.IndexFunc(l.Files, func(f madeFile) bool { return f.Path == orchardTo })
		if i >= 0 {
			made = l.Files[i]
		}
		return i >= 0, nil
	})
	if made.PlacedBy != "converted" || made.MadeFrom == nil || made.MadeFrom.Name != filepath.Base(placed) || made.Media.Video.Codec != "h264" || made.Media.Video.Height != 720 {
		t.Fatalf("the made file is %+v, want placed by converted from %s, H.264 at 720p", made, filepath.Base(placed))
	}
	if _, err := os.Stat(filepath.Join(m.MediaDir, "movies", filepath.FromSlash(orchardTo))); err != nil {
		t.Fatalf("the made file isn't in the movies root: %v", err)
	}
}
