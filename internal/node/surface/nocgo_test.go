package surface_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestTheSurfaceNeverReachesLibfvad(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", ".")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps without cgo: %v\n%s", err, out)
	}
	deps := strings.Fields(string(out))
	if !slices.Contains(deps, "github.com/jatassi/slipstream/internal/storing") {
		t.Fatalf("the surface's deps leave out internal/storing, so this test proves nothing:\n%s", out)
	}
	if slices.Contains(deps, "github.com/jatassi/slipstream/internal/media/fvad") {
		t.Error("the surface reaches internal/media/fvad, so the API document needs cgo to build")
	}
}
