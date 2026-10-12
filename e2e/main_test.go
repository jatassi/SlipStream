// Package e2e is the end-to-end tier: tests that launch a Fake world
// against the binary under test and drive it as a client does.
package e2e

import (
	"os"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
)

func TestMain(m *testing.M) {
	os.Exit(testbin.Run(m))
}
