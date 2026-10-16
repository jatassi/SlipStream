package node

import (
	"os"
	"testing"
)

func TestFixture(t *testing.T) {
	_ = os.WriteFile(t.TempDir()+"/config.yaml", nil, 0o600)
}
