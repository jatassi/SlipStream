package formatsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// dated reads a name's words up to its year, bracketed or not, as its
// title. A name with no year reads no title. Its clean rule folds case.
type dated struct{}

func (dated) ReadName(p release.Prepared) release.Name {
	words := strings.FieldsFunc(p.Title.String(), func(r rune) bool { return strings.ContainsRune(". _", r) })
	for i, w := range words {
		y, err := strconv.Atoi(strings.Trim(w, "()"))
		if err == nil && y > 1900 && i > 0 {
			return release.Name{Titles: []string{strings.Join(words[:i], " ")}, Year: release.Known(y)}
		}
	}
	return release.Name{}
}

func (dated) CleanTitle(title string) string { return strings.ToLower(title) }

// claims expects a Work's label as its one title and 2020 as its year,
// and every name to cover its first Item.
type claims struct{ w *works.Works }

func (c claims) Expect(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) (indexers.Expect, error) {
	tree, err := c.w.TreeIn(ctx, r, work)
	if err != nil {
		return indexers.Expect{}, err
	}
	return indexers.Expect{Claim: release.Claim{Titles: []string{tree.Label}, Year: release.Known(2020), Slack: 1}, Covers: indexers.CoversOnly(tree.Items[0].ID)}, nil
}

func (claims) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

func (claims) Categories() []newznab.ID { return nil }

type parsedName struct {
	Sentence string `json:"sentence"`
	Words    string `json:"words"`
	Works    []struct {
		Label string `json:"label"`
	} `json:"works"`
	Covers  []string `json:"covers"`
	Formats []struct {
		Name    string `json:"name"`
		Draft   bool   `json:"draft"`
		Matched bool   `json:"matched"`
	} `json:"formats"`
	Targets []struct {
		Name      string `json:"name"`
		Score     int    `json:"score"`
		WouldFill bool   `json:"wouldFill"`
		Reasons   []struct {
			Code  string `json:"code"`
			Value string `json:"value"`
		} `json:"reasons"`
	} `json:"targets"`
}

// parse is each name's result from POST /releases/parse with body.
func (s *served) parse(t *testing.T, body string) []parsedName {
	t.Helper()
	code, reply := s.Call(t, http.MethodPost, "/api/v1/releases/parse", body)
	var got struct {
		Results []parsedName `json:"results"`
	}
	if code != http.StatusOK || json.Unmarshal([]byte(reply), &got) != nil {
		t.Fatalf("parseReleases = %d %s", code, reply)
	}
	return got.Results
}

func (p parsedName) labels() string {
	var out []string
	for _, w := range p.Works {
		out = append(out, w.Label)
	}
	return strings.Join(out, ", ")
}

// codes is the reasons at p's first Target, as code=value.
func (p parsedName) codes() string {
	var out []string
	for _, r := range p.Targets[0].Reasons {
		out = append(out, r.Code+"="+r.Value)
	}
	return strings.Join(out, " ")
}

func TestTheTesterJudgesEachNameAgainstTheGivenWorksHeldFile(t *testing.T) {
	s := serve(t)
	got := s.parse(t, `{"names":["The.Attic.2020.1080p.BluRay.x264-GRP","The.Attic.2020.1080p.WEB-DL-GRP","The.Cellar.2019.1080p.BluRay-GRP"],"workId":"`+s.attic+`"}`)
	for i, want := range []struct{ sentence, works, codes string }{
		{"Reads as Bluray-1080p from GRP. It matches The Attic and would fill 1080p.", "The Attic", "size_not_checked="},
		{"Reads as WEBDL-1080p from GRP. It matches The Attic and would fill no Target. It isn't an upgrade of the file at 1080p.", "The Attic", "size_not_checked= not_an_upgrade="},
		{"Reads as Bluray-1080p from GRP. It matches no Work and would fill no Target. Title doesn't match: The Cellar (2019).", "", "wrong_title=The Cellar (2019) size_not_checked="},
	} {
		if got[i].Sentence != want.sentence || got[i].labels() != want.works || got[i].codes() != want.codes || len(got[i].Covers) != 1 {
			t.Errorf("name %d = %q, works %q, reasons %q, covers %v\nwant %q, works %q, reasons %q", i, got[i].Sentence, got[i].labels(), got[i].codes(), got[i].Covers, want.sentence, want.works, want.codes)
		}
	}
}

func TestTheTesterMatchesANameToTheWorkWhoseTitleAndYearItGives(t *testing.T) {
	s := serve(t)
	got := s.parse(t, `{"names":["The.Attic.2020.1080p.BluRay.x264-GRP"]}`)[0]
	if want := "Reads as Bluray-1080p from GRP. It matches The Attic and would fill 1080p."; got.Sentence != want || got.labels() != "The Attic" || len(got.Covers) != 1 {
		t.Errorf("matched = %q, works %q, covers %v, want %q", got.Sentence, got.labels(), got.Covers, want)
	}
	if got.Words != "Bluray-1080p · GRP" {
		t.Errorf("words = %q", got.Words)
	}
}

func TestANameThatMatchesNoWorkIsJudgedAtEveryTargetWithNoHeldFile(t *testing.T) {
	s := serve(t)
	got := s.parse(t, `{"names":["Nothing.2020.2160p.WEB-DL-GRP","The.Attic.1990.1080p.WEB-DL-GRP"]}`)
	for i, want := range []struct{ sentence, codes string }{
		{"Reads as WEBDL-2160p from GRP. It matches no Work and would fill no Target. 1080p doesn't accept WEBDL-2160p.", "quality_not_in_target=webdl-2160p"},
		{"Reads as WEBDL-1080p from GRP. It matches no Work and would fill 1080p.", "size_not_checked="},
	} {
		if got[i].Sentence != want.sentence || len(got[i].Works) != 0 || len(got[i].Covers) != 0 || got[i].codes() != want.codes {
			t.Errorf("name %d = %q, works %q, covers %v, reasons %q\nwant %q, reasons %q", i, got[i].Sentence, got[i].labels(), got[i].Covers, got[i].codes(), want.sentence, want.codes)
		}
	}
}

func TestTheTesterScoresTheFormatsANameMatchesWithADraftInPlaceOfItsSavedFormat(t *testing.T) {
	s := serve(t)
	group := func(name, pattern string) string {
		return `{"name":"` + name + `","conditions":[{"name":"Group","kind":"ReleaseGroupSpecification","negate":false,"required":false,"fields":{"value":"` + pattern + `"}}]}`
	}
	_, body := s.Call(t, http.MethodPost, "/api/v1/custom-formats", group("Good group", "^GRP$"))
	id := ownertest.FirstID(body)
	s.scoreStarter(t, id)
	names := `{"names":["The.Attic.2020.1080p.BluRay.x264-GRP"],"workId":"` + s.attic + `"`
	formatsOf := func(p parsedName) string {
		var out []string
		for _, f := range p.Formats {
			out = append(out, f.Name+" draft="+strconv.FormatBool(f.Draft)+" matched="+strconv.FormatBool(f.Matched))
		}
		return strings.Join(out, ", ")
	}
	for _, c := range []struct {
		name, draft, formats string
		score                int
	}{
		{"the saved format", ``, "Good group draft=false matched=true", 25},
		{"a draft of it that no longer matches", `,"draft":{"formatId":"` + id + `",` + group("Good group", "^OTHER$")[1:], "Good group draft=true matched=false", 0},
		{"a new draft beside it", `,"draft":` + group("Also good", "GRP"), "Also good draft=true matched=true, Good group draft=false matched=true", 25},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := s.parse(t, names+c.draft+`}`)[0]
			if formatsOf(got) != c.formats || got.Targets[0].Score != c.score {
				t.Errorf("formats %q, score %d, want %q, score %d", formatsOf(got), got.Targets[0].Score, c.formats, c.score)
			}
		})
	}
	detail := `The pattern in \"Group\" doesn't compile (unterminated group). Fix the pattern and try it again.`
	s.expect(t, http.MethodPost, "/api/v1/releases/parse", names+`,"draft":`+group("Broken", "(a")+`}`, http.StatusBadRequest,
		`{"status":400,"code":"format_pattern_not_compiled","title":"The pattern in this condition doesn't compile","detail":"`+detail+
			`","errors":[{"location":"body.draft.conditions[0].fields.value","message":"`+detail+`"}]}`)
}

func TestAFileNameReadsWithItsFolderWhenItGivesNoTitle(t *testing.T) {
	s := serve(t)
	path := `D:\\Films\\The Attic (2020)\\1080p.BluRay-GRP.mkv`
	asFile := s.parse(t, `{"names":["`+path+`"],"kind":"file"}`)[0]
	if want := "Reads as Bluray-1080p from GRP. It matches The Attic and would fill 1080p."; asFile.Sentence != want {
		t.Errorf("kind file = %q, want %q", asFile.Sentence, want)
	}
	asRelease := s.parse(t, `{"names":["`+path+`"]}`)[0]
	if asRelease.labels() != "" {
		t.Errorf("kind release matches %q", asRelease.labels())
	}
}
