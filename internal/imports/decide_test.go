package imports

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func cellOf(target targets.TargetID) files.CellKey {
	return files.CellKey{Item: works.ItemID{UUID: storage.NewUUID()}, Edition: targets.AnyEdition, Target: target}
}

func given(src Source, othersFill bool) outcomeContext {
	return outcomeContext{
		othersFill: othersFill, source: src,
		cut:       func(int) release.CutKey { return "directors" },
		confirmed: func(int) bool { return false },
		forced:    func(int) []files.CellKey { return nil },
	}
}

func TestOutcomesFollowTheSpecsDecision(t *testing.T) {
	hd := targets.TargetID{UUID: storage.NewUUID()}
	cell := cellOf(hd)
	fills := files.Verdict{Fills: []files.CellKey{cell}, Meets: []files.CellKey{cell}}
	lostToLive := files.Verdict{Meets: []files.CellKey{cell}, Lost: map[files.CellKey]files.Holder{cell: files.HeldByLive}}
	lostToSibling := files.Verdict{Meets: []files.CellKey{cell}, Lost: map[files.CellKey]files.Holder{cell: files.HeldBySibling}}
	meetsNothing := files.Verdict{Reasons: map[targets.TargetID]targets.Verdict{hd: {}}}
	for _, tc := range []struct {
		name string
		vs   []files.Verdict
		c    outcomeContext
		want []Outcome
		why  []ReasonCode
	}{
		{"a file that fills a cell goes on to placing", []files.Verdict{fills}, given(FromRoot, false), []Outcome{Placing}, []ReasonCode{""}},
		{"a file of another Node goes on to sending", []files.Verdict{fills}, func() outcomeContext { c := given(FromGrab, false); c.sends = true; return c }(), []Outcome{Sending}, []ReasonCode{""}},
		{"a manual import's file waits for its confirm", []files.Verdict{fills}, given(FromFolder, false), []Outcome{Waiting}, []ReasonCode{Confirm}},
		{"a file whose cells a sibling fills is left out", []files.Verdict{fills, lostToSibling}, given(FromGrab, false), []Outcome{Placing, LeftOut}, []ReasonCode{"", SiblingBetter}},
		{"a file that fills nothing while others fill something isn't better", []files.Verdict{fills, lostToLive}, given(FromGrab, false), []Outcome{Placing, NotBetter}, []ReasonCode{"", KeptBetter}},
		{"a file outside the batch filling something counts", []files.Verdict{lostToLive}, given(FromGrab, true), []Outcome{NotBetter}, []ReasonCode{KeptBetter}},
		{"when nothing fills anything every file waits not_an_upgrade", []files.Verdict{lostToLive, lostToLive}, given(FromGrab, false), []Outcome{Waiting, Waiting}, []ReasonCode{NotAnUpgrade, NotAnUpgrade}},
		{"a file meeting no Target is left out", []files.Verdict{fills, meetsNothing}, given(FromGrab, false), []Outcome{Placing, LeftOut}, []ReasonCode{"", MeetsNothing}},
		{"the only video meeting no Target waits", []files.Verdict{meetsNothing}, given(FromGrab, false), []Outcome{Waiting}, []ReasonCode{MeetsNothing}},
		{"a cut no wanted Edition takes waits", []files.Verdict{{EditionGone: true}}, given(FromGrab, false), []Outcome{Waiting}, []ReasonCode{EditionNotWanted}},
		{"a taken name waits", []files.Verdict{{Refusal: &files.NameTakenError{Path: "Heat (1995).mkv"}}}, given(FromRoot, false), []Outcome{Waiting}, []ReasonCode{NameTaken}},
		{"a name too long waits", []files.Verdict{{Refusal: &naming.TooLongError{Bytes: 291, Limit: 259}}}, given(FromRoot, false), []Outcome{Waiting}, []ReasonCode{NameTooLong}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []Outcome
			var why []ReasonCode
			for _, o := range outcomes(tc.vs, tc.c) {
				got = append(got, o.To)
				if o.Reason == nil {
					why = append(why, "")
				} else {
					why = append(why, o.Reason.Code)
				}
			}
			if !slices.Equal(got, tc.want) || !slices.Equal(why, tc.why) {
				t.Errorf("outcomes = %v %v, want %v %v", got, why, tc.want, tc.why)
			}
		})
	}
}

func TestAForcedCellLandsForced(t *testing.T) {
	hd, uhd := targets.TargetID{UUID: storage.NewUUID()}, targets.TargetID{UUID: storage.NewUUID()}
	a, b := cellOf(hd), cellOf(uhd)
	c := given(FromGrab, false)
	c.forced = func(int) []files.CellKey { return []files.CellKey{b} }
	o := outcomes([]files.Verdict{{Fills: []files.CellKey{a, b}, Meets: []files.CellKey{a, b}}}, c)[0]
	if want := []Fill{{Cell: a}, {Cell: b, Forced: true}}; !slices.Equal(o.Fills, want) {
		t.Errorf("fills %v, want %v", o.Fills, want)
	}
}

func TestEveryReasonOutcomesGivesIsWorded(t *testing.T) {
	for _, code := range []ReasonCode{NameTaken, NameTooLong, EditionNotWanted, MeetsNothing, Confirm, SiblingBetter, KeptBetter, NotAnUpgrade} {
		if _, ok := rules[code]; !ok {
			t.Errorf("outcomes gives %s, which no rule words", code)
		}
	}
}
