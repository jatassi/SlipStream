package main

import (
	"slices"
	"testing"
)

func TestSplitTierFlag(t *testing.T) {
	for _, c := range []struct {
		args, tier, rest []string
	}{
		{nil, []string{}, []string{}},
		{[]string{"-tier", "go", "-run", "Foo", "./e2e"}, []string{"-tier", "go"}, []string{"-run", "Foo", "./e2e"}},
		{[]string{"--tier=browser"}, []string{"--tier=browser"}, []string{}},
		{[]string{"-run", "Foo", "-tier", "go"}, []string{}, []string{"-run", "Foo", "-tier", "go"}},
		{[]string{"-tier"}, []string{"-tier"}, []string{}},
	} {
		tier, rest := splitTierFlag(c.args)
		if !slices.Equal(tier, c.tier) || !slices.Equal(rest, c.rest) {
			t.Errorf("splitTierFlag(%q) = %q, %q, want %q, %q", c.args, tier, rest, c.tier, c.rest)
		}
	}
}

func TestTestRefusesGoArgumentsOutsideTheGoTier(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"-run", "Foo"}, `go test arguments ["-run" "Foo"] need -tier go`},
		{[]string{"-tier", "component", "./..."}, `go test arguments ["./..."] need -tier go`},
		{[]string{"-tier", "e2e"}, `-tier "e2e" is not one of go, component, browser, all`},
	} {
		if err := runTest(t.Context(), c.args); err == nil || err.Error() != c.want {
			t.Errorf("runTest(%q) = %v, want %s", c.args, err, c.want)
		}
	}
}
