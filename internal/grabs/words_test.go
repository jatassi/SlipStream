package grabs_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
)

func TestEveryCodeHasWords(t *testing.T) {
	nothingToDo := map[grabs.Code]bool{grabs.TransferFailed: true, grabs.ImportFailed: true, grabs.Declined: true}
	for _, c := range grabs.Codes {
		w := grabs.WordsOf(c)
		if w.Cause == "" {
			t.Errorf("%s has no cause", c)
		}
		if (w.Recovery == "") != nothingToDo[c] {
			t.Errorf("%s has recovery %q", c, w.Recovery)
		}
	}
}
