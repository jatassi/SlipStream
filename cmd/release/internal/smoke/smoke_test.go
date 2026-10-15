package smoke

import (
	"slices"
	"strings"
	"testing"
)

func TestCommandRemovesSlipStreamVariables(t *testing.T) {
	t.Setenv("SLIPSTREAM_NATIVE_CACHE", "/runner/cache")
	t.Setenv("SLIPSTREAM_VERBOSE", "1")
	t.Setenv("SMOKE_KEPT", "yes")
	cmd := command(t.Context(), []string{"DISPLAY=:99"}, "slipstream", "app")
	if i := slices.IndexFunc(cmd.Env, func(kv string) bool { return strings.HasPrefix(kv, "SLIPSTREAM_") }); i >= 0 {
		t.Errorf("the command's environment holds %q", cmd.Env[i])
	}
	for _, want := range []string{"SMOKE_KEPT=yes", "DISPLAY=:99"} {
		if !slices.Contains(cmd.Env, want) {
			t.Errorf("the command's environment lacks %q", want)
		}
	}
}
