package e2e

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const (
	dv8Film       = "Lanterns at Low Tide (2016) {tmdb-9000101}/Lanterns at Low Tide (2016).mkv"
	hdr10PlusFilm = "Saltmarsh Radio (2018)/Saltmarsh Radio (2018).mkv"
	sdrEp         = "The Tin Signal [tvdbid-9100201]/Season 01/The Tin Signal - S01E01.mkv"
	hdr10Ep       = "The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E05.mkv"
	hlgEp         = "The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E06.mkv"
	dv5Ep         = "Brass Meridian [tvdbid-9100202]/Brass Meridian - 2x05.mkv"
)

type listedMedia struct {
	Path  string `json:"path"`
	Kept  bool   `json:"kept"`
	Fills []struct {
		TargetID string `json:"targetId"`
	} `json:"fills"`
	Media *struct {
		Video struct {
			Codec       string          `json:"codec"`
			BitDepth    int             `json:"bitDepth"`
			HDR         []string        `json:"hdr"`
			DolbyVision json.RawMessage `json:"dolbyVision"`
		} `json:"video"`
		Audio     json.RawMessage `json:"audio"`
		Subtitles json.RawMessage `json:"subtitles"`
	} `json:"media"`
}

// scannedHome adds home's shelves as roots under a series Profile that
// holds only an HDR10 Target, and returns each scanned file by path with
// the Work it joined. Brass Meridian's folder carries its tvdb id, so its
// files reach a Work by tag rather than by a title search.
func scannedHome(t *testing.T) (a *worldtest.API, listed map[string]listedMedia, works map[string]string) {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "home-mediainfo"
	for i, f := range s.Library.Folders {
		for j, m := range f.Files {
			s.Library.Folders[i].Files[j].Path = strings.Replace(m.Path, "Brass Meridian (2015)/", "Brass Meridian [tvdbid-9100202]/", 1)
		}
	}
	s.Library.Targets = []scenario.LibraryTarget{{
		Name: "HDR10", Tiers: [][]scenario.QualityKey{{"hdtv-1080p"}}, Cutoff: "hdtv-1080p",
		Filters: []scenario.LibraryFilter{{Attribute: "hdr", Mode: "only", Values: []string{"HDR10"}}},
	}}
	s.Library.Profiles = []scenario.LibraryProfile{{Name: "HDR", ContentType: "series", Targets: []string{"HDR10"}, Default: true}}
	run := worldtest.LaunchScenario(t, s)
	a = api(t, run)
	addShelves(t, a, run.Manifest())
	worldtest.Until(t, "the first scans list every probed file on its Work", func() (bool, error) {
		var library struct {
			Rows []struct {
				ID string `json:"id"`
			} `json:"rows"`
		}
		decode(t, "getLibrary", a.Admin("home", http.MethodGet, "/api/v1/library", ""), &library)
		listed, works = map[string]listedMedia{}, map[string]string{}
		for _, w := range library.Rows {
			var files struct {
				Files []listedMedia `json:"files"`
			}
			decode(t, "listWorkFiles", a.Admin("home", http.MethodGet, "/api/v1/works/"+w.ID+"/files", ""), &files)
			for _, f := range files.Files {
				listed[f.Path], works[f.Path] = f, w.ID
			}
		}
		for _, p := range []string{dv8Film, hdr10PlusFilm, sdrEp, hdr10Ep, hlgEp, dv5Ep} {
			if listed[p].Media == nil {
				return false, nil
			}
		}
		return true, nil
	})
	return a, listed, works
}

func TestTheWorkFilesReadEveryHDRFormatAndEachTracksLanguage(t *testing.T) {
	t.Parallel()
	_, listed, _ := scannedHome(t)
	type read struct{ video, dolbyVision, audio, subtitles string }
	want := map[string]read{
		dv8Film: {
			"hevc 10-bit DV,HDR10", `{"profile":8,"level":3,"compatibility":1}`,
			`[{"codec":"truehd","profile":"","atmos":false,"channels":6,"layout":"5.1(side)","language":"en","default":true,"title":""},` +
				`{"codec":"aac","profile":"LC","atmos":false,"channels":2,"layout":"stereo","language":"en","default":false,"title":"Commentary"}]`,
			`[{"codec":"subrip","image":false,"language":"en","forced":false,"default":false}]`,
		},
		hdr10PlusFilm: {
			"hevc 10-bit HDR10+,HDR10", "null",
			`[{"codec":"eac3","profile":"","atmos":false,"channels":6,"layout":"5.1(side)","language":"en","default":true,"title":""},` +
				`{"codec":"eac3","profile":"","atmos":false,"channels":6,"layout":"5.1(side)","language":"fr","default":false,"title":""}]`,
			`[{"codec":"subrip","image":false,"language":"en","forced":false,"default":false},{"codec":"subrip","image":false,"language":"fr","forced":true,"default":false}]`,
		},
		sdrEp: {
			"h264 8-bit SDR", "null",
			`[{"codec":"aac","profile":"LC","atmos":false,"channels":2,"layout":"stereo","language":"en","default":false,"title":""}]`, `[]`,
		},
		hdr10Ep: {
			"hevc 10-bit HDR10", "null",
			`[{"codec":"eac3","profile":"","atmos":false,"channels":6,"layout":"5.1(side)","language":"en","default":true,"title":""}]`, `[]`,
		},
		hlgEp: {
			"hevc 10-bit HLG", "null",
			`[{"codec":"eac3","profile":"","atmos":false,"channels":6,"layout":"5.1(side)","language":"en","default":true,"title":""}]`, `[]`,
		},
		dv5Ep: {
			"hevc 10-bit DV", `{"profile":5,"level":3,"compatibility":0}`,
			`[{"codec":"flac","profile":"","atmos":false,"channels":2,"layout":"stereo","language":"en","default":true,"title":""},` +
				`{"codec":"aac","profile":"LC","atmos":false,"channels":2,"layout":"stereo","language":"de","default":false,"title":""}]`, `[]`,
		},
	}
	for p, w := range want {
		m := listed[p].Media
		got := read{
			m.Video.Codec + " " + strconv.Itoa(m.Video.BitDepth) + "-bit " + strings.Join(m.Video.HDR, ","),
			string(m.Video.DolbyVision), string(m.Audio), string(m.Subtitles),
		}
		if got != w {
			t.Errorf("%s reads\n%+v\nwant\n%+v", p, got, w)
		}
	}
}

func TestAnHDR10TargetJudgesTheHDR10FileMetAndTheSDRFileNot(t *testing.T) {
	t.Parallel()
	a, listed, works := scannedHome(t)
	if works[sdrEp] != works[hdr10Ep] {
		t.Fatalf("the SDR and HDR10 episodes joined Works %s and %s, want both on The Tin Signal", works[sdrEp], works[hdr10Ep])
	}
	hdr10, sdr := listed[hdr10Ep], listed[sdrEp]
	if hdr10.Kept || len(hdr10.Fills) != 1 || !sdr.Kept || len(sdr.Fills) != 0 {
		t.Fatalf("the HDR10 file reads kept=%v fills=%+v and the SDR file kept=%v fills=%+v; want the HDR10 file filling one cell and the SDR file Kept",
			hdr10.Kept, hdr10.Fills, sdr.Kept, sdr.Fills)
	}
	cells := decode(t, "getWorkCells", a.Admin("home", http.MethodGet, "/api/v1/works/"+works[hdr10Ep]+"/cells", ""), &filledCells{})
	var got []string
	for _, c := range cells.Cells {
		line := c.TargetName + " " + c.State
		if c.File != nil {
			line += " " + c.File.Path
		}
		got = append(got, line)
	}
	want := []string{"HDR10 missing", "HDR10 missing", "HDR10 met " + hdr10Ep, "HDR10 missing"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("The Tin Signal's cells read\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
