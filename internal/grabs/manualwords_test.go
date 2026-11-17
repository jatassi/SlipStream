package grabs

import (
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestEveryBlocklistCauseReadsMidSentenceAfterBecause(t *testing.T) {
	t.Parallel()
	for _, c := range BlockCauses {
		clause := becauseClause(BlockEntry{Cause: c, Client: "qBittorrent", By: &Actor{}})
		if first, _ := utf8.DecodeRuneInString(clause); unicode.IsUpper(first) {
			t.Errorf("the %s cause reads %q after \"because\", want it to start in lower case", c, clause)
		}
	}
}
