package grabs_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/grabs"
)

func TestOnlyAFailureTheAdminDidntChooseNeedsThem(t *testing.T) {
	for _, c := range []struct {
		code   grabs.Code
		source grabs.Source
		want   bool
	}{
		{grabs.DownloadFailed, grabs.FromSearch, true},
		{grabs.ReleaseFault, grabs.FromManual, true},
		{grabs.Canceled, grabs.FromSearch, false},
		{grabs.Canceled, grabs.FromManual, false},
		{grabs.Blocklisted, grabs.FromSearch, true},
		{grabs.Blocklisted, grabs.FromManual, false},
	} {
		g := grabs.Grab{Stage: grabs.Failed, Failed: &grabs.Failure{Code: c.code}, Source: c.source}
		if got := g.NeedsYou(); got != c.want {
			t.Errorf("a %s Grab failed %s needs the Admin: %v, want %v", c.source, c.code, got, c.want)
		}
	}
	if (grabs.Grab{Stage: grabs.Downloading, Source: grabs.FromSearch}).NeedsYou() {
		t.Error("a downloading Grab needs the Admin")
	}
}
