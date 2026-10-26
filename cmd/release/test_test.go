package main

import (
	"flag"
	"slices"
	"testing"
)

func TestSplitOwnFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("tier", "", "")
	fs.String("shard", "", "")
	for _, c := range []struct {
		args, own, rest []string
	}{
		{nil, []string{}, []string{}},
		{[]string{"-tier", "go", "-run", "Foo", "./e2e"}, []string{"-tier", "go"}, []string{"-run", "Foo", "./e2e"}},
		{[]string{"--tier=browser", "-shard", "2/4"}, []string{"--tier=browser", "-shard", "2/4"}, []string{}},
		{[]string{"-run", "Foo", "-tier", "go"}, []string{}, []string{"-run", "Foo", "-tier", "go"}},
		{[]string{"-tier"}, []string{"-tier"}, []string{}},
	} {
		own, rest := splitOwnFlags(fs, c.args)
		if !slices.Equal(own, c.own) || !slices.Equal(rest, c.rest) {
			t.Errorf("splitOwnFlags(%q) = %q, %q, want %q, %q", c.args, own, rest, c.own, c.rest)
		}
	}
}

func TestTestRefusesFlagsTheTierDoesNotTake(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"-run", "Foo"}, `go test arguments ["-run" "Foo"] need -tier go, e2e or online`},
		{[]string{"-tier", "component", "./..."}, `go test arguments ["./..."] need -tier go, e2e or online`},
		{[]string{"-tier", "lint"}, `-tier "lint" is not one of go, e2e, component, diagram, browser, online, all`},
		{[]string{"-shard", "1/2"}, "-shard, -filter and -repeat need -tier go, e2e or browser"},
		{[]string{"-tier", "component", "-filter", "Foo"}, "-shard, -filter and -repeat need -tier go, e2e or browser"},
		{[]string{"-tier", "browser", "-timings", "t"}, "-timings and -record need -tier go or e2e"},
		{[]string{"-tier", "go", "-shard", "1/2", "./internal/..."}, `go test arguments ["./internal/..."] pick the tests themselves, so they take none of -shard, -filter, -repeat, -timings or -record`},
		{[]string{"-tier", "go", "-repeat", "0"}, "-repeat 0 is not a positive count"},
		{[]string{"-tier", "go", "-shard", "3/2"}, `parse flags: invalid value "3/2" for flag -shard: shard "3/2" is not i/n with 1 <= i <= n`},
	} {
		if err := runTest(t.Context(), c.args); err == nil || err.Error() != c.want {
			t.Errorf("runTest(%q) = %v, want %s", c.args, err, c.want)
		}
	}
}

func TestPlaywrightRuns(t *testing.T) {
	for _, c := range []struct {
		o    testOptions
		want [][]string
	}{
		{testOptions{shard: shard{2, 3}, filter: "@phone", repeat: 5}, [][]string{
			{"--grep-invert=@alone", "--pass-with-no-tests", "--shard=2/3", "--repeat-each=5", "--grep=@phone"},
			{"--workers=1", "--grep=(?=.*@alone)(?=.*(?:@phone))", "--pass-with-no-tests", "--shard=2/3", "--repeat-each=5"},
		}},
		{testOptions{repeat: 1}, [][]string{
			{"--grep-invert=@alone", "--pass-with-no-tests"},
			{"--workers=1", "--grep=@alone", "--pass-with-no-tests"},
		}},
	} {
		got := playwrightRuns(c.o)
		if !slices.EqualFunc(got, c.want, slices.Equal) {
			t.Errorf("playwrightRuns(%+v) = %q, want %q", c.o, got, c.want)
		}
	}
}
