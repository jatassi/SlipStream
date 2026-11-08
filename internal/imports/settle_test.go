package imports

import (
	"testing"

	"github.com/jatassi/slipstream/internal/scanning"
)

func TestAnUnsureFolderSuggestsOnlyACandidateItsReasonNames(t *testing.T) {
	t.Parallel()
	cs := []scanning.Candidate{
		{Key: "1", Titles: []string{"The Quiet Orchard"}, Year: 2021},
		{Key: "2", Titles: []string{"Halloween"}, Year: 1978},
		{Key: "3", Titles: []string{"Halloween"}, Year: 2018},
		{Key: "4", Titles: []string{"Brazil"}, Year: 1985},
	}
	for _, tt := range []struct {
		hint      scanning.Hint
		code      scanning.ReasonCode
		suggested int
	}{
		{scanning.Hint{Title: "a"}, scanning.NoMatch, -1},
		{scanning.Hint{Title: "Halloween"}, scanning.Several, 1},
		{scanning.Hint{Title: "Brazil"}, scanning.NoYear, 3},
		{scanning.Hint{Title: "Brazil", Year: 1986}, scanning.NoMatch, -1},
	} {
		r, named := unsure(tt.hint, cs)
		if r.Code != tt.code || named != tt.suggested {
			t.Errorf("a folder hinting %+v waits for %s suggesting %d, want %s suggesting %d", tt.hint, r.Code, named, tt.code, tt.suggested)
		}
	}
}
