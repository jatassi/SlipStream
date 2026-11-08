package targets_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

var update = flag.Bool("update", false, "rewrite testdata/upgrades.json and testdata/filters.json from Judge and the vocabularies")

// The web app's Target editor says in words which moves a Target allows.
// It reads these files in its own tests, so its rules and Judge can't
// drift.
type move struct {
	From      *side    `json:"from"`
	To        side     `json:"to"`
	Grabbable bool     `json:"grabbable"`
	Reasons   []string `json:"reasons"`
	Rule      int      `json:"rule"`
}

// side is a release or held file. Version 0 reads as 1, and an empty
// group is unknown.
type side struct {
	Quality string `json:"quality"`
	Version int    `json:"version,omitempty"`
	Real    int    `json:"real,omitempty"`
	Repack  bool   `json:"repack,omitempty"`
	Group   string `json:"group,omitempty"`
	Score   int    `json:"score,omitempty"`
}

func (s side) facts(t *testing.T) release.Facts {
	t.Helper()
	f := release.Facts{
		Quality:  release.Known(must[release.Quality](t)(release.ParseQuality(s.Quality))),
		Revision: release.Revision{Version: max(s.Version, 1), Real: s.Real, Repack: s.Repack},
	}
	if s.Group != "" {
		f.Group = release.Known(s.Group)
	}
	return f
}

type judged struct {
	Name   string
	Tiers  [][]string
	Cutoff string
	Scores struct {
		Minimum      int `json:"minimum"`
		UpgradeUntil int `json:"upgradeUntil"`
		Increment    int `json:"increment"`
	}
	Propers string
	Moves   []move
}

func layout(t *testing.T, name string, p targets.Preset, chosen func(release.QualityInfo) bool, upgrades bool) targets.Target {
	t.Helper()
	var qs []release.Quality
	for _, q := range release.Qualities() {
		if chosen(q.Info()) {
			qs = append(qs, q)
		}
	}
	s := targets.Layout(p, qs, upgrades)
	s.Name = name
	return must[targets.Target](t)(targets.NewTarget(s))
}

// offer is a release a golden judges: a side and the formats it matches.
type offer struct {
	side
	formats formats.Set
}

func judgeMoves(t *testing.T, tg targets.Target, helds []*side, offers []offer) judged {
	t.Helper()
	s := tg.Spec()
	out := judged{Name: tg.Name, Cutoff: s.Cutoff.String(), Propers: tg.Propers.String()}
	out.Scores.Minimum, out.Scores.UpgradeUntil, out.Scores.Increment = tg.Scores.Minimum, tg.Scores.UpgradeUntil, tg.Scores.Increment
	for _, tier := range s.Tiers {
		out.Tiers = append(out.Tiers, names(tier))
	}
	for _, from := range helds {
		var held *targets.Held
		if from != nil {
			held = &targets.Held{Facts: from.facts(t), Score: from.Score}
		}
		for _, o := range offers {
			v := targets.Judge(tg, targets.Scored{Facts: o.facts(t), Formats: o.formats}, held)
			reasons := []string{}
			for _, r := range v.Reasons() {
				if r.Code != targets.SizeNotChecked {
					reasons = append(reasons, string(r.Code))
				}
			}
			to := o.side
			to.Score = v.Score()
			rule, _ := v.Upgrade()
			out.Moves = append(out.Moves, move{From: from, To: to, Grabbable: v.Grabbable(), Reasons: reasons, Rule: rule})
		}
	}
	return out
}

// judgeEveryMove judges every quality over every quality held, and over
// none.
func judgeEveryMove(t *testing.T, tg targets.Target) judged {
	t.Helper()
	helds := []*side{nil}
	var offers []offer
	for _, q := range release.Qualities() {
		helds = append(helds, &side{Quality: q.String()})
		offers = append(offers, offer{side: side{Quality: q.String()}})
	}
	return judgeMoves(t, tg, helds, offers)
}

func names[V fmt.Stringer](vs []V) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.String()
	}
	return out
}

func values[V ~string](vs []V) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

// encode writes one move per line, so a change to Judge reads as a diff of
// the moves it changed.
func encode(t *testing.T, all []judged) []byte {
	t.Helper()
	var b bytes.Buffer
	b.WriteString("{\"targets\": [\n")
	for i, j := range all {
		head, err := json.Marshal(struct {
			Cutoff  string     `json:"cutoff"`
			Name    string     `json:"name"`
			Propers string     `json:"propers"`
			Scores  any        `json:"scores"`
			Tiers   [][]string `json:"tiers"`
		}{j.Cutoff, j.Name, j.Propers, j.Scores, j.Tiers})
		if err != nil {
			t.Fatal(err)
		}
		b.Write(head[:len(head)-1])
		b.WriteString(", \"moves\": [\n")
		for k, m := range j.Moves {
			b.WriteString("  ")
			b.Write(must[[]byte](t)(json.Marshal(m)))
			if k < len(j.Moves)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString("]}")
		if i < len(all)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]}\n")
	return b.Bytes()
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; run go test ./internal/targets -run TestWebGoldens -update", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s is stale; run go test ./internal/targets -run TestWebGoldens -update and the web tests", path)
	}
}

func TestWebGoldens(t *testing.T) {
	hd := func(i release.QualityInfo) bool {
		return i.Resolution == release.R720p || i.Resolution == release.R1080p
	}
	notCinema := func(i release.QualityInfo) bool { return !i.Cinema() }
	every := func(release.QualityInfo) bool { return true }
	middle := layout(t, "Anything better", targets.AnythingBetter, notCinema, true).Spec()
	middle.Cutoff = release.WEBDL1080p
	scored := func(name string, minimum, until, increment int) targets.Target {
		s := layout(t, name, targets.DiscOrResolution, hd, true).Spec()
		s.Scores = targets.Scores{Formats: map[formats.ID]int{fa: 10, fb: 25, fc: -10}, Minimum: minimum, UpgradeUntil: until, Increment: increment}
		return must[targets.Target](t)(targets.NewTarget(s))
	}
	scoredHelds := []*side{nil}
	for _, q := range []string{"webdl-1080p", "bluray-1080p"} {
		for _, score := range []int{-10, 0, 10, 30, 40} {
			scoredHelds = append(scoredHelds, &side{Quality: q, Score: score})
		}
	}
	var scoredOffers []offer
	for _, q := range []string{"webdl-1080p", "webrip-1080p", "remux-1080p", "bluray-720p"} {
		for _, fs := range []formats.Set{nil, {fa}, {fb}, {fa, fb}, {fc}} {
			scoredOffers = append(scoredOffers, offer{side: side{Quality: q}, formats: fs})
		}
	}
	propers := func(name string, p targets.Propers) targets.Target {
		s := layout(t, name, targets.DiscOrResolution, hd, true).Spec()
		s.Propers = p
		return must[targets.Target](t)(targets.NewTarget(s))
	}
	revisions := []side{{}, {Version: 2}, {Version: 2, Repack: true}, {Real: 1}}
	var propersHelds []*side
	for _, q := range []string{"webdl-1080p", "bluray-1080p"} {
		for _, r := range revisions[:2] {
			for _, g := range []string{"FLUX", ""} {
				propersHelds = append(propersHelds, &side{Quality: q, Version: r.Version, Group: g})
			}
		}
	}
	var propersOffers []offer
	for _, q := range []string{"webdl-1080p", "webrip-1080p", "bluray-1080p"} {
		for _, r := range revisions {
			for _, g := range []string{"flux", "NTb", ""} {
				propersOffers = append(propersOffers, offer{side: side{Quality: q, Version: r.Version, Real: r.Real, Repack: r.Repack, Group: g}})
			}
		}
	}
	golden(t, "upgrades.json", encode(t, []judged{
		judgeEveryMove(t, layout(t, "1080p", targets.DiscOrResolution, hd, true)),
		judgeEveryMove(t, layout(t, "No upgrades", targets.DiscOrResolution, hd, false)),
		judgeEveryMove(t, must[targets.Target](t)(targets.NewTarget(middle))),
		judgeEveryMove(t, layout(t, "Resolution only", targets.ResolutionOnly, every, true)),
		judgeMoves(t, scored("Scored", -5, 30, 10), scoredHelds, scoredOffers),
		judgeMoves(t, scored("Scored, upgrade-until 0", 0, 0, 1), scoredHelds, scoredOffers),
		judgeMoves(t, propers("Prefer and upgrade", targets.PreferAndUpgrade), propersHelds, propersOffers),
		judgeMoves(t, propers("Prefer, don't upgrade", targets.PreferNoUpgrade), propersHelds, propersOffers),
		judgeMoves(t, propers("Don't prefer", targets.DontPrefer), propersHelds, propersOffers),
	}))
	if !slices.Equal(targets.Attributes()[:4], []targets.Attribute{targets.HDR, targets.VideoCodec, targets.AudioCodec, targets.Channels}) {
		t.Fatalf("the attributes are %v", targets.Attributes())
	}
	vocab := map[string][]string{
		string(targets.HDR):        values(release.HDRs()),
		string(targets.VideoCodec): values(release.VideoCodecs()),
		string(targets.AudioCodec): values(release.AudioCodecs()),
		string(targets.Channels):   values(release.ChannelLayouts()),
	}
	golden(t, "filters.json", append(must[[]byte](t)(json.MarshalIndent(vocab, "", "  ")), '\n'))
}
