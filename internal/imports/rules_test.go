package imports

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// specTable is a markdown table of testdata/spec_reasons.txt: each row's
// cells, its header left out.
type specTable [][]string

// specReasons reads the fixture: the example words, then the release-fault
// table and the waiting table.
func specReasons(t *testing.T) (words map[string]string, faults, waits specTable) {
	t.Helper()
	b, err := os.ReadFile("testdata/spec_reasons.txt")
	if err != nil {
		t.Fatal(err)
	}
	words = map[string]string{}
	var tables []specTable
	for line := range strings.Lines(string(b)) {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "words: "):
			for pair := range strings.SplitSeq(strings.TrimPrefix(line, "words: "), "; ") {
				k, v, _ := strings.Cut(pair, "=")
				words[k] = v
			}
		case strings.HasPrefix(line, "|---"):
			tables = append(tables, nil)
		case strings.HasPrefix(line, "|") && len(tables) > 0:
			var cells []string
			for c := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
				cells = append(cells, strings.TrimSpace(c))
			}
			tables[len(tables)-1] = append(tables[len(tables)-1], cells)
		}
	}
	if len(tables) != 2 {
		t.Fatalf("the fixture holds %d tables, want the release-fault table and the waiting table", len(tables))
	}
	return words, tables[0], tables[1]
}

// exampleWording words every reason with the fixture's examples.
type exampleWording map[string]string

func (w exampleWording) targets([]targets.TargetID) string { return w["target"] }
func (w exampleWording) items(ids []works.ItemID) string {
	switch len(ids) {
	case 1:
		return w["reading"]
	case 3:
		return w["busy"]
	}
	return w["of"]
}
func (w exampleWording) level(works.LevelID) string    { return w["level"] }
func (w exampleWording) edition(release.CutKey) string { return w["edition"] }
func (w exampleWording) item() string                  { return w["item"] }
func (w exampleWording) work() string                  { return w["work"] }
func (exampleWording) label() string                   { return "" }

// exampleReasons are the waiting table's reasons with the values its
// sentences show.
var exampleReasons = map[ReasonCode]Reason{
	NotAnUpgrade:     {Code: NotAnUpgrade},
	MeetsNothing:     {Code: MeetsNothing},
	EditionNotWanted: {Code: EditionNotWanted},
	OtherItems:       {Code: OtherItems, Reading: make([]works.ItemID, 1), Of: make([]works.ItemID, 4)},
	NoNumber:         {Code: NoNumber},
	Parts:            {Code: Parts, Parts: 2},
	NameTaken:        {Code: NameTaken, Name: "Heat (1995).mkv"},
	NameTooLong:      {Code: NameTooLong, Length: 291, Limit: 259},
	Settings:         {Code: Settings, Setting: SmallestVideo.Key, Value: 50 << 20},
	Busy:             {Code: Busy, Reading: make([]works.ItemID, 3)},
}

// actionWords reads the spec's action words.
func actionWords(words string) []ActionKind {
	var out []ActionKind
	for w := range strings.SplitSeq(words, ", ") {
		switch {
		case w == "none":
		case w == "Don't import":
			out = append(out, ActDecline)
		case strings.HasPrefix(w, "Force into"):
			out = append(out, ActForce)
		case w == "Not this Release":
			out = append(out, ActNotThisRelease)
		case w == "Change the Work's Editions":
			out = append(out, ActChangeEditions)
		case strings.HasPrefix(w, "Import as"):
			out = append(out, ActConfirm)
		case w == "Change the match":
			out = append(out, ActMatch)
		case w == "Retry":
			out = append(out, ActRetry)
		case w == "Turn on renaming":
			out = append(out, ActTurnOnRenaming)
		case w == "Change the setting":
			out = append(out, ActChangeSetting)
		default:
			out = append(out, ActionKind("unknown: "+w))
		}
	}
	return out
}

func TestEverySpecReasonHasItsSentenceAndAction(t *testing.T) {
	words, _, waits := specReasons(t)
	grab := Import{Source: FromGrab}
	for _, row := range waits {
		code := ReasonCode(strings.Trim(row[0], "`"))
		r, ok := exampleReasons[code]
		if !ok {
			t.Errorf("%s: the spec's waiting reason has no example here", code)
			continue
		}
		want := strings.Trim(row[1], `"`)
		got := r.sentence(exampleWording(words))
		if code == MeetsNothing {
			// The spec's second sentence is the first Target's verdict, which the row's Details word.
			want, _, _ = strings.Cut(want, ". ")
			want += "."
		}
		if got != want {
			t.Errorf("%s: sentence %q, want %q", code, got, want)
		}
		f := File{Outcome: Waiting, Reason: &r}
		recommended, also := f.Actions(grab)
		wantRec := actionWords(row[2])
		if got := slices.DeleteFunc([]ActionKind{recommended}, func(a ActionKind) bool { return a == "" }); !slices.Equal(got, wantRec) {
			t.Errorf("%s: recommends %v, want %v", code, got, wantRec)
		}
		if want := actionWords(row[3]); !slices.Equal(also, want) {
			t.Errorf("%s: also offers %v, want %v", code, also, want)
		}
	}
	if len(waits) != len(exampleReasons) {
		t.Errorf("the spec lists %d waiting reasons and the examples %d", len(waits), len(exampleReasons))
	}
}

func TestNotThisReleaseIsOfferedOnlyOnAGrabsImport(t *testing.T) {
	f := File{Outcome: Waiting, Reason: &Reason{Code: NotAnUpgrade}}
	for src, want := range map[Source][]ActionKind{
		FromGrab:   {ActForce, ActNotThisRelease},
		FromFolder: {ActForce, ActMatch},
		FromRoot:   {ActForce},
	} {
		if _, also := f.Actions(Import{Source: src}); !slices.Equal(also, want) {
			t.Errorf("a %s Import's not_an_upgrade file also offers %v, want %v", src, also, want)
		}
	}
}

func TestEveryActionIsTheWiresInItsOrder(t *testing.T) {
	seen := map[ActionKind]bool{}
	for _, rl := range rules {
		if rl.recommended != "" {
			seen[rl.recommended] = true
		}
		for _, a := range rl.also {
			seen[a] = true
		}
	}
	for a := range seen {
		if !slices.Contains(ActionKinds, a) {
			t.Errorf("a rule offers %s, which ActionKinds doesn't list", a)
		}
	}
}

func TestEveryOfferedHoldOffersRetryAndDontImport(t *testing.T) {
	f := File{Outcome: Pending, Hold: &Hold{Code: ProbeTimeout, Attempts: 3}}
	if rec, also := f.Actions(Import{Source: FromGrab}); rec != ActRetry || !slices.Equal(also, []ActionKind{ActDecline}) {
		t.Errorf("three probe timeouts offer %s and %v, want Retry and Don't import", rec, also)
	}
	f.Hold.Attempts = 2
	if rec, also := f.Actions(Import{Source: FromGrab}); rec != "" || also != nil {
		t.Errorf("two probe timeouts offer %s and %v, want nothing yet", rec, also)
	}
}

// wireEnums reads the reason codes and the actions the published OpenAPI
// document lists.
func wireEnums(t *testing.T) (reasons, actions []string) {
	t.Helper()
	b, err := os.ReadFile("../../web/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas struct {
				Action struct {
					Enum []string `json:"enum"`
				} `json:"ImportsAction"`
				Reason struct {
					Properties struct {
						Code struct {
							Enum []string `json:"enum"`
						} `json:"code"`
					} `json:"properties"`
				} `json:"ImportsReason"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Components.Schemas.Reason.Properties.Code.Enum, doc.Components.Schemas.Action.Enum
}

func TestTheReasonTableIsPinnedToTheWire(t *testing.T) {
	reasons, actions := wireEnums(t)
	for code := range rules {
		if !slices.Contains(reasons, string(code)) {
			t.Errorf("the rule for %s words a reason the wire's reason enum doesn't list", code)
		}
	}
	wire := make([]ActionKind, len(actions))
	for i, a := range actions {
		wire[i] = ActionKind(a)
	}
	if !slices.Equal(wire, ActionKinds) {
		t.Errorf("the wire's actions are %v, want ActionKinds %v", wire, ActionKinds)
	}
}
