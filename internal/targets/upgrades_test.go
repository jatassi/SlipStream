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

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

var update = flag.Bool("update", false, "rewrite testdata/upgrades.json and testdata/filters.json from Judge and the vocabularies")

// The web app's Target editor says in words which moves a Target allows.
// It reads these files in its own tests, so its rule and Judge can't drift.
type move struct {
	From      *string  `json:"from"`
	To        string   `json:"to"`
	Grabbable bool     `json:"grabbable"`
	Reasons   []string `json:"reasons"`
}

type judged struct {
	Name   string
	Tiers  [][]string
	Cutoff string
	Moves  []move
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

func judgeEveryMove(tg targets.Target) judged {
	s := tg.Spec()
	out := judged{Name: tg.Name, Cutoff: s.Cutoff.String()}
	for _, tier := range s.Tiers {
		out.Tiers = append(out.Tiers, names(tier))
	}
	helds := []*release.Facts{nil}
	for _, q := range release.Qualities() {
		helds = append(helds, &release.Facts{Quality: release.Known(q)})
	}
	for _, held := range helds {
		var from *string
		if held != nil {
			q, _ := held.Quality.Get()
			key := q.String()
			from = &key
		}
		for _, to := range release.Qualities() {
			v := targets.Judge(tg, release.Facts{Quality: release.Known(to)}, held)
			reasons := []string{}
			for _, r := range v.Reasons() {
				if r.Code != targets.SizeNotChecked {
					reasons = append(reasons, string(r.Code))
				}
			}
			out.Moves = append(out.Moves, move{From: from, To: to.String(), Grabbable: v.Grabbable(), Reasons: reasons})
		}
	}
	return out
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
			Cutoff string     `json:"cutoff"`
			Name   string     `json:"name"`
			Tiers  [][]string `json:"tiers"`
		}{j.Cutoff, j.Name, j.Tiers})
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
	golden(t, "upgrades.json", encode(t, []judged{
		judgeEveryMove(layout(t, "1080p", targets.DiscOrResolution, hd, true)),
		judgeEveryMove(layout(t, "No upgrades", targets.DiscOrResolution, hd, false)),
		judgeEveryMove(must[targets.Target](t)(targets.NewTarget(middle))),
		judgeEveryMove(layout(t, "Resolution only", targets.ResolutionOnly, every, true)),
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
