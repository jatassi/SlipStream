package main_test

import (
	"os"
	"testing"

	"github.com/jatassi/slipstream/fakes/testbin"
)

func TestMain(m *testing.M) {
	os.Exit(testbin.Run(m))
}
