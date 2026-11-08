package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
)

const (
	flux        = "The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.H264-FLUX"
	freeleech   = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-CMRG"
	repack      = "The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.H264-FKW"
	otherRepack = "The.Quiet.Orchard.2021.REPACK.1080p.WEB-DL.H264-NTb"
	otherYear   = "The.Quiet.Orchard.1987.1080p.BluRay.x264-FKW"
)

// listedRelease is one Release a run saw, as listSearchReleases lists it.
type listedRelease struct {
	runRelease
	Scores []struct {
		TargetID string `json:"targetId"`
		Score    int    `json:"score"`
	} `json:"scores"`
	LostAt *struct {
		Step string `json:"step"`
		Own  string `json:"own"`
		Ref  string `json:"ref"`
	} `json:"lostAt"`
}

// String is "<title> <outcome>", then where it lost, its scores and its
// reasons' codes and values.
func (r listedRelease) String() string {
	line := r.Title + " " + r.Outcome
	if r.LostAt != nil {
		line += fmt.Sprintf(" lost at %s %s<%s", r.LostAt.Step, r.LostAt.Own, r.LostAt.Ref)
	}
	for _, s := range r.Scores {
		line += fmt.Sprintf(" score %d", s.Score)
	}
	for _, reason := range r.Reasons {
		line += " " + reason.Code
		if reason.Value != "" {
			line += "=" + reason.Value
		}
	}
	return line
}

// releasesAre checks the Releases run id saw, in the order listed, are
// want.
func (w *walked) releasesAre(t *testing.T, id string, want ...string) {
	t.Helper()
	var list struct {
		Releases []listedRelease `json:"releases"`
	}
	decode(t, "listSearchReleases", w.a.Admin("home", http.MethodGet, "/api/v1/searches/"+id+"/releases", ""), &list)
	var got []string
	for _, r := range list.Releases {
		got = append(got, r.String())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("run %s saw the Releases\n%s\nwant\n%s", id, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// grabbedOf searches and checks the run grabbed of candidates and its
// Grab imported. It answers the run's id.
func (w *walked) grabbedOf(t *testing.T, candidates int) string {
	t.Helper()
	id := w.search(t)
	found := w.searched(t, id)
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.Candidates != candidates || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a grab of %d candidates", found, candidates)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	return id
}

// pasteTRaSH pastes TRaSH Guides' Radarr format file as its text, for
// Radarr, and answers the id of the format it adds.
func (w *walked) pasteTRaSH(t *testing.T, file string) string {
	t.Helper()
	text, err := os.ReadFile("../internal/formats/testdata/trash/radarr/cf/" + file)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{"text": string(text), "app": "radarr"})
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		App     string `json:"app"`
		Formats []struct {
			Outcome string `json:"outcome"`
			SavedAs string `json:"savedAs"`
		} `json:"formats"`
		Impact impactView `json:"impact"`
	}
	decode(t, "pasteCustomFormats", w.a.Admin("home", http.MethodPost, "/api/v1/custom-formats/paste", string(body)), &p)
	if p.App != "radarr" || len(p.Formats) != 1 || p.Formats[0].Outcome != "add" || p.Impact.Works != 0 {
		t.Fatalf("pasting %s answered %+v, want one format added for Radarr with no Impact", file, p)
	}
	var list struct {
		Formats []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"formats"`
	}
	decode(t, "listCustomFormats", w.a.Admin("home", http.MethodGet, "/api/v1/custom-formats", ""), &list)
	if len(list.Formats) != 1 || list.Formats[0].Name != p.Formats[0].SavedAs {
		t.Fatalf("listCustomFormats lists %+v, want the pasted %s alone", list.Formats, p.Formats[0].SavedAs)
	}
	return list.Formats[0].ID
}

// score sets Target id's scores to scores, a newScores body, keeping the
// rest of the Target, and checks the Impact it answers is want.
func (w *walked) score(t *testing.T, id, scores, want string) {
	t.Helper()
	var held struct {
		Name    string          `json:"name"`
		Tiers   json.RawMessage `json:"tiers"`
		Cutoff  string          `json:"cutoff"`
		Filters json.RawMessage `json:"filters"`
		Sizes   json.RawMessage `json:"sizes"`
		Propers string          `json:"propers"`
	}
	decode(t, "getTarget", w.a.Admin("home", http.MethodGet, "/api/v1/targets/"+id, ""), &held)
	body, err := json.Marshal(map[string]any{
		"name": held.Name, "tiers": held.Tiers, "cutoff": held.Cutoff, "filters": held.Filters, "sizes": held.Sizes,
		"propers": held.Propers, "scores": json.RawMessage(scores),
	})
	if err != nil {
		t.Fatal(err)
	}
	w.put(t, "/api/v1/targets/"+id, string(body), want)
}

func TestAPastedTRaSHFormatScoredAtATargetChangesWhichReleaseASearchGrabs(t *testing.T) {
	t.Parallel()
	w := launchedTargets(t)
	profile := "/api/v1/profiles/" + w.named(t, "/api/v1/profiles", "profiles", "Films")
	starter := w.named(t, "/api/v1/targets", "targets", "1080p")
	w.put(t, profile, films(starter), "works=0 ownList=0 unwanted=0 renamed=0 newlyWanted=0 downloading=0")
	format := w.pasteTRaSH(t, "flux.json")
	w.list(t, flux)
	w.add(t)

	unscored := w.grabbedOf(t, 2)
	w.releasesAre(t, unscored,
		release+" grabbed score 0",
		flux+" outranked lost at seeders 3<12 score 0",
	)
	w.holds(t, release)

	w.score(t, starter, `{"formats":[{"formatId":"`+format+`","score":51}],"upgradeUntil":51,"increment":1}`,
		"works=0 ownList=0 unwanted=0 renamed=0 newlyWanted=0 downloading=0")
	scored := w.grabbedOf(t, 2)
	w.releasesAre(t, scored,
		flux+" grabbed score 51",
		release+" refused score 0 not_an_upgrade",
	)
	w.holds(t, flux)
}

func TestAReleaseWhoseNameGivesAnotherYearIsRefusedAsWrongTitle(t *testing.T) {
	t.Parallel()
	w := launchedTargets(t)
	w.list(t, otherYear)
	w.add(t)
	id := w.grabbedOf(t, 2)
	w.releasesAre(t, id,
		otherYear+" refused score 0 score 0 wrong_title=The Quiet Orchard (1987)",
		release+" grabbed score 0 score 0",
	)
	w.holds(t, release)
}

func TestARepackFromTheHeldFilesGroupReplacesItAndOneFromAnotherGroupDoesnt(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	_, held := w.cells(t).summary(t)
	w.list(t, repack)
	w.list(t, otherRepack)

	id := w.grabbedOf(t, 3)
	w.releasesAre(t, id,
		repack+" grabbed score 0 score 0",
		release+" refused score 0 score 0 not_an_upgrade not_an_upgrade",
		otherRepack+" refused score 0 score 0 repack_other_group repack_other_group",
	)
	w.holds(t, repack)
	name := "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	got, file := w.cells(t).summary(t)
	if want := []string{"1080p upgradable wanted webdl-1080p " + name, "HD upgradable wanted webdl-1080p " + name}; !slices.Equal(got, want) || file == held {
		t.Fatalf("after the repack the cells are %q, want %q in a new file", got, want)
	}
}

func TestAFreeleechReleaseOutranksOneWithMoreSeeders(t *testing.T) {
	t.Parallel()
	w := launchedTargets(t)
	w.list(t, freeleech)
	w.add(t)
	id := w.grabbedOf(t, 2)
	w.releasesAre(t, id,
		freeleech+" grabbed score 0 score 0",
		release+" outranked lost at freeleech 1<0 score 0 score 0",
	)
	w.holds(t, freeleech)
}

// tried is one name as parseReleases answers it.
type tried struct {
	Sentence string `json:"sentence"`
	Words    string `json:"words"`
	Facts    struct {
		Quality  string   `json:"quality"`
		Group    string   `json:"group"`
		HDR      []string `json:"hdr"`
		Video    string   `json:"video"`
		Audio    []string `json:"audio"`
		Channels []string `json:"channels"`
		Year     int      `json:"year"`
	} `json:"facts"`
	Works []struct {
		Label string `json:"label"`
	} `json:"works"`
	Formats []struct {
		Name       string `json:"name"`
		Matched    bool   `json:"matched"`
		Conditions []struct {
			Name    string `json:"name"`
			Matched bool   `json:"matched"`
		} `json:"conditions"`
	} `json:"formats"`
	Targets []struct {
		Name      string `json:"name"`
		Score     int    `json:"score"`
		WouldFill bool   `json:"wouldFill"`
	} `json:"targets"`
}

// lines is the name as lines a reader compares: its sentence, words and
// facts, then each Work, format with its conditions, and Target.
func (r tried) lines() []string {
	f := r.Facts
	out := []string{r.Sentence, r.Words, fmt.Sprintf("%s %s %v %s %v %v %d", f.Quality, f.Group, f.HDR, f.Video, f.Audio, f.Channels, f.Year)}
	for _, w := range r.Works {
		out = append(out, "work "+w.Label)
	}
	for _, c := range r.Formats {
		line := fmt.Sprintf("format %s %t", c.Name, c.Matched)
		for _, cond := range c.Conditions {
			line += fmt.Sprintf(" %s %t", cond.Name, cond.Matched)
		}
		out = append(out, line)
	}
	for _, t := range r.Targets {
		out = append(out, fmt.Sprintf("target %s %d %t", t.Name, t.Score, t.WouldFill))
	}
	return out
}

func TestTheTesterReadsNamesAndJudgesThemAtEachTargetThroughTheAPI(t *testing.T) {
	t.Parallel()
	w := launchedTargets(t)
	w.add(t)
	starter := w.named(t, "/api/v1/targets", "targets", "1080p")
	format := w.pasteTRaSH(t, "flux.json")
	w.score(t, starter, `{"formats":[{"formatId":"`+format+`","score":51}],"upgradeUntil":51,"increment":1}`,
		"works=0 ownList=0 unwanted=0 renamed=0 newlyWanted=0 downloading=0")
	dv := "The.Quiet.Orchard.2021.1080p.WEB-DL.DDP5.1.Atmos.DV.HDR10.x265-FLUX"
	draft := `{"name":"Not FKW","conditions":[{"name":"FKW","kind":"ReleaseGroupSpecification","negate":true,"required":false,"fields":{"value":"^(FKW)$"}}]}`
	var parsed struct {
		Results []tried `json:"results"`
	}
	decode(t, "parseReleases", w.a.Admin("home", http.MethodPost, "/api/v1/releases/parse", `{"names":["`+dv+`","`+otherYear+`"],"draft":`+draft+`}`), &parsed)
	want := [][]string{{
		"Reads as WEBDL-1080p from FLUX. It matches The Quiet Orchard and would fill 1080p and HD.",
		"WEBDL-1080p · Dolby Vision · DD+ 5.1 · FLUX",
		"webdl-1080p FLUX [DV HDR10] x265 [DD+] [5.1] 2021",
		"work The Quiet Orchard",
		"format FLUX true FLUX true",
		"format Not FKW true FKW true",
		"target 1080p 51 true",
		"target HD 0 true",
	}, {
		"Reads as Bluray-1080p from FKW. It matches no Work and would fill 1080p and HD.",
		"Bluray-1080p · FKW",
		"bluray-1080p FKW [] x264 [] [] 1987",
		"format FLUX false FLUX false",
		"format Not FKW false FKW false",
		"target 1080p 0 true",
		"target HD 0 true",
	}}
	if len(parsed.Results) != len(want) {
		t.Fatalf("parseReleases answered %d results, want %d", len(parsed.Results), len(want))
	}
	for i, r := range parsed.Results {
		if got := r.lines(); !slices.Equal(got, want[i]) {
			t.Fatalf("the tester read name %d as\n%s\nwant\n%s", i, strings.Join(got, "\n"), strings.Join(want[i], "\n"))
		}
	}
}
