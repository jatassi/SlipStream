package grabs

import (
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestEveryReasonReadsAsASentence(t *testing.T) {
	at := targets.Target{Name: "1080p"}
	for _, code := range targets.ReasonCodes() {
		words := ReasonWords(Reason{Reason: targets.Reason{Code: code, Value: "x"}}, at, release.Facts{})
		if !strings.HasSuffix(words, ".") || strings.ContainsAny(words, "—–") {
			t.Errorf("%s reads %q", code, words)
		}
	}
}
