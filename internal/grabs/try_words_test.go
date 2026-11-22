package grabs

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var shelf = works.Description{Work: works.Noun{One: "Shelf", Many: "Shelves"}, Item: works.Noun{One: "Thing", Many: "Things"}}

func TestEveryReasonReadsAsASentence(t *testing.T) {
	for _, at := range []*targets.Target{{Name: "1080p"}, nil} {
		for _, code := range targets.ReasonCodes() {
			words := reasonWords(Reason{Reason: targets.Reason{Code: code, Value: "x"}}, at, release.Facts{}, shelf)
			if !strings.HasSuffix(words, ".") || strings.ContainsAny(words, "—–") {
				t.Errorf("%s reads %q", code, words)
			}
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
		if got := reasonWords(Reason{Reason: targets.Reason{Code: targets.RepackOtherGroup, Value: c.held}}, &targets.Target{Name: "1080p"}, c.facts, shelf); got != c.want {
			t.Errorf("held %q reads %q, want %q", c.held, got, c.want)
		}
	}
}
