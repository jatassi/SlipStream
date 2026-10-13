package main

import (
	"os"
	"strings"
	"testing"
)

// mainArgs, set in a child's environment, runs the release main with
// these arguments instead of the tests, so a test sees its exit code.
const mainArgs = "RELEASE_TEST_MAIN_ARGS"

func TestMain(m *testing.M) {
	if args, ok := os.LookupEnv(mainArgs); ok {
		os.Args = append([]string{"release"}, strings.Fields(args)...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}
