package grabs

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestEveryReasonReadsAsASentence(t *testing.T) {
	for _, at := range []*targets.Target{{Name: "1080p"}, nil} {
		for _, code := range targets.ReasonCodes() {
			words := reasonWords(Reason{Reason: targets.Reason{Code: code, Value: "x"}}, at, release.Facts{})
			if !strings.HasSuffix(words, ".") || strings.ContainsAny(words, "—–") {
				t.Errorf("%s reads %q", code, words)
			}
		}
	}
}

func TestAReasonAtADeletedTargetNamesItAsDeleted(t *testing.T) {
	for code, want := range map[targets.ReasonCode]string{
		targets.QualityNotInTarget: "A deleted Target doesn't accept a release with no Quality.",
		targets.ScoreBelowMinimum:  "Score 10 is below the minimum of a deleted Target.",
		targets.SizeAbove:          "It's larger than a deleted Target allows.",
	} {
		if got := reasonWords(Reason{Reason: targets.Reason{Code: code, Value: "10"}}, nil, release.Facts{}); got != want {
			t.Errorf("%s reads %q, want %q", code, got, want)
		}
	}
}

func TestARepackFromAnotherGroupNamesBothGroupsAndTheTarget(t *testing.T) {
	ntb := release.Facts{Group: release.Known("NTb")}
	for _, c := range []struct {
		held  string
		facts release.Facts
		want  string
	}{
		{"FKW", ntb, "It's a repack from NTb, and the file at 1080p is from FKW."},
		{"", ntb, "It's a repack from NTb, and the file at 1080p has no release group."},
		{"FKW", release.Facts{}, "It's a repack with no release group, and the file at 1080p is from FKW."},
	} {
		if got := reasonWords(Reason{Reason: targets.Reason{Code: targets.RepackOtherGroup, Value: c.held}}, &targets.Target{Name: "1080p"}, c.facts); got != c.want {
			t.Errorf("held %q reads %q, want %q", c.held, got, c.want)
		}
	}
}
