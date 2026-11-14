package main

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func packages(paths ...string) []goPackage {
	pkgs := make([]goPackage, len(paths))
	for i, p := range paths {
		pkgs[i] = goPackage{path: p, tests: true}
	}
	return pkgs
}

// placed counts, by package, the shards that run it whole.
func placed(shards [][]goRun) map[string]int {
	count := map[string]int{}
	for _, runs := range shards {
		for _, r := range runs {
			if r.run != "" || r.skip != "" {
				continue
			}
			for _, p := range r.packages {
				count[p]++
			}
		}
	}
	return count
}

func TestPartitionPacksByTestFileSizeWithoutTimings(t *testing.T) {
	pkgs := []goPackage{
		{path: "cmd/release", tests: true, testBytes: 58},
		{path: "cmd/slipstream", tests: true, testBytes: 107},
		{path: "e2e", tests: true, testBytes: 329},
		{path: "fakes/tracker", tests: true, testBytes: 63},
		{path: "fakes/world/sim", tests: true, testBytes: 230},
		{path: "notests"},
	}
	shards := partition(pkgs, timings{}, 2, true)
	count := placed(shards)
	if len(count) != len(pkgs) || slices.ContainsFunc(slices.Collect(maps.Values(count)), func(n int) bool { return n != 1 }) {
		t.Fatalf("packages placed %v, want each of %d once", count, len(pkgs))
	}
	shardOf := map[string]int{}
	for i, runs := range shards {
		for _, r := range runs {
			for _, p := range r.packages {
				shardOf[p] = i + 1
			}
		}
	}
	if shardOf["e2e"] == shardOf["fakes/world/sim"] {
		t.Errorf("e2e and fakes/world/sim both run on shard %d of %v, want the two largest test files apart", shardOf["e2e"], shards)
	}
}

func TestPartitionPacksLongestFirst(t *testing.T) {
	past := timings{Packages: map[string]packageTimings{
		"a": {Seconds: 10}, "b": {Seconds: 6}, "c": {Seconds: 5}, "d": {Seconds: 4}, "e": {Seconds: 3},
	}}
	shards := partition(packages("a", "b", "c", "d", "e"), past, 2, true)
	got := []string{strings.Join(shards[0][0].packages, " "), strings.Join(shards[1][0].packages, " ")}
	if want := []string{"a d", "b c e"}; !slices.Equal(got, want) {
		t.Errorf("shards = %q, want %q", got, want)
	}
}

// go test starts a shard's packages in the order it lists them, so a long
// package listed last starts only when the others free the CPUs.
func TestPartitionListsAShardsPackagesLongestFirst(t *testing.T) {
	past := timings{Packages: map[string]packageTimings{"a": {Seconds: 1}, "m": {Seconds: 5}, "z": {Seconds: 10}}}
	shards := partition(packages("a", "m", "z"), past, 1, true)
	if got, want := shards[0][0].packages, []string{"z", "m", "a"}; !slices.Equal(got, want) {
		t.Errorf("shard lists %q, want %q", got, want)
	}
}

func TestPartitionWeighsUnseenPackagesAsTheMean(t *testing.T) {
	past := timings{Packages: map[string]packageTimings{"a": {Seconds: 9}, "b": {Seconds: 3}}}
	shards := partition(packages("a", "b", "new"), past, 2, true)
	got := []string{strings.Join(shards[0][0].packages, " "), strings.Join(shards[1][0].packages, " ")}
	if want := []string{"a", "new b"}; !slices.Equal(got, want) {
		t.Errorf("shards = %q, want %q", got, want)
	}
}

func TestPartitionSplitsAHeavyPackageByItsTests(t *testing.T) {
	past := timings{Packages: map[string]packageTimings{
		"heavy": {Seconds: 90, Tests: map[string]float64{"TestA": 40, "TestB": 30, "TestC": 10, "TestD": 5, "TestE": 5}},
		"x":     {Seconds: 10},
		"y":     {Seconds: 10},
		"z":     {Seconds: 10},
	}}
	pkgs := packages("heavy", "x", "y", "z")
	shards := partition(pkgs, past, 3, true)
	if count := placed(shards); count["heavy"] != 0 || count["x"] != 1 || count["y"] != 1 || count["z"] != 1 {
		t.Fatalf("packages placed whole %v, want x, y and z once and heavy split", count)
	}
	parts := splitParts(t, shards)
	if len(parts) != 3 {
		t.Fatalf("heavy splits into %v, want 3 parts", parts)
	}
	for _, name := range append(slices.Collect(maps.Keys(past.Packages["heavy"].Tests)), "TestNew") {
		if n := timesRun(parts, name); n != 1 {
			t.Errorf("the parts of heavy run %s %d times, want once", name, n)
		}
	}
	if whole := partition(pkgs, past, 3, false); placed(whole)["heavy"] != 1 {
		t.Errorf("without splitTests heavy is placed %d times whole, want once", placed(whole)["heavy"])
	}
}

func TestPartitionSpreadsTestsTheTimingsHaveNotSeen(t *testing.T) {
	timed := map[string]float64{"TestA": 10, "TestB": 10, "TestC": 10, "TestD": 10}
	past := timings{Packages: map[string]packageTimings{"heavy": {Seconds: 40, Tests: timed}, "x": {Seconds: 1}}}
	pkgs := packages("heavy", "x")
	pkgs[0].testNames = []string{"TestA", "TestB", "TestC", "TestD", "TestN1", "TestN2", "TestN3", "TestN4"}
	parts := splitParts(t, partition(pkgs, past, 2, true))
	if len(parts) != 2 {
		t.Fatalf("heavy splits into %v, want 2 parts", parts)
	}
	for _, name := range pkgs[0].testNames {
		if n := timesRun(parts, name); n != 1 {
			t.Errorf("the parts of heavy run %s %d times, want once", name, n)
		}
	}
	for _, p := range parts {
		var fresh int
		for _, name := range pkgs[0].testNames[4:] {
			if timesRun([]goRun{p}, name) == 1 {
				fresh++
			}
		}
		if fresh != 2 {
			t.Errorf("part %v runs %d of the 4 untimed tests, want 2", p.args(), fresh)
		}
	}
}

func TestPartitionSplitsAPackageLongerThanAShardCanSpread(t *testing.T) {
	past := timings{CPUs: 4, Packages: map[string]packageTimings{
		"node": {Seconds: 144, Tests: map[string]float64{"TestA": 50, "TestB": 50, "TestC": 44}},
	}}
	pkgs := packages("node")
	for i := range 99 {
		p := fmt.Sprintf("p%02d", i)
		past.Packages[p] = packageTimings{Seconds: 4}
		pkgs = append(pkgs, goPackage{path: p, tests: true})
	}
	shards := partition(pkgs, past, 2, true)
	if count := runCounts(shards); count["node"] != 1 || placed(shards)["node"] != 0 {
		t.Errorf("node, 144 s of 540 on two 4-CPU shards, runs %v, want it split across both", shards)
	}
	if past.CPUs = 1; placed(partition(pkgs, past, 2, true))["node"] != 1 {
		t.Error("node, within one shard's share on one CPU, was split")
	}
}

// splitParts lists the parts of split packages across shards, and fails
// the test when a shard runs two parts of one package.
func splitParts(t *testing.T, shards [][]goRun) []goRun {
	t.Helper()
	var parts []goRun
	for i, runs := range shards {
		seen := map[string]bool{}
		for _, r := range runs {
			if r.run == "" && r.skip == "" {
				continue
			}
			if seen[r.packages[0]] {
				t.Errorf("shard %d runs two parts of %s", i+1, r.packages[0])
			}
			seen[r.packages[0]] = true
			parts = append(parts, r)
		}
	}
	return parts
}

// timesRun counts the parts whose -run matches the test, or whose -skip
// does not.
func timesRun(parts []goRun, test string) int {
	n := 0
	for _, p := range parts {
		if p.run != "" && regexp.MustCompile(p.run).MatchString(test) || p.skip != "" && !regexp.MustCompile(p.skip).MatchString(test) {
			n++
		}
	}
	return n
}

func TestListGoPackagesSumsTheTestFiles(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := listGoPackages(t.Context(), root, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "cmd", "release", "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	var want int64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			t.Fatal(err)
		}
		want += info.Size()
	}
	i := slices.IndexFunc(pkgs, func(p goPackage) bool { return p.path == "github.com/jatassi/slipstream/cmd/release" })
	if i < 0 {
		t.Fatal("go list leaves out cmd/release")
	}
	if pkgs[i].testBytes != want {
		t.Errorf("cmd/release test files weigh %d bytes, want the %d its *_test.go files hold", pkgs[i].testBytes, want)
	}
	if !slices.Contains(pkgs[i].testNames, "TestListGoPackagesSumsTheTestFiles") {
		t.Errorf("cmd/release declares tests %v, want this one among them", pkgs[i].testNames)
	}
}

// TestGoAndE2ETiersRunEveryPackageOnce partitions the module's packages as
// the go and e2e tiers' shards do, for each shard count the lanes use,
// and checks that together they run every package exactly once.
func TestGoAndE2ETiersRunEveryPackageOnce(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := listGoPackages(t.Context(), root, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "go", "list", "./...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Fields(string(out))

	var e2e, goTier []goPackage
	for _, p := range pkgs {
		if p.e2e {
			e2e = append(e2e, p)
		} else {
			goTier = append(goTier, p)
		}
	}
	for _, want := range []string{"cmd/fakeworld", "cmd/release", "cmd/slipstream", "e2e", "fakes/world"} {
		if !slices.ContainsFunc(e2e, func(p goPackage) bool { return p.path == "github.com/jatassi/slipstream/"+want }) {
			t.Errorf("the e2e tier leaves out %s, whose tests drive binaries through fakes/testbin", want)
		}
	}
	past := timings{Packages: map[string]packageTimings{}}
	for _, p := range pkgs {
		past.Packages[p.path] = packageTimings{Seconds: 1}
	}
	heavy := goTier[slices.IndexFunc(goTier, func(p goPackage) bool { return p.tests })].path
	past.Packages[heavy] = packageTimings{Seconds: 1e6, Tests: map[string]float64{"TestA": 1, "TestB": 1, "TestC": 1}}
	for n := 1; n <= 4; n++ {
		goShards := partition(goTier, past, n, true)
		count := runCounts(goShards)
		for p, c := range runCounts(partition(e2e, past, n, true)) {
			count[p] += c
		}
		if _, whole := placed(goShards)[heavy]; n > 1 && whole {
			t.Errorf("%d shards run %s whole, want it split by its tests", n, heavy)
		}
		if len(count) != len(all) || slices.ContainsFunc(all, func(p string) bool { return count[p] != 1 }) {
			t.Errorf("%d shards run packages %v, want each of %v once", n, count, all)
		}
	}
}

// runCounts counts, by package, the shards that run it whole, and once
// more for a package split across shards.
func runCounts(shards [][]goRun) map[string]int {
	count := placed(shards)
	split := map[string]bool{}
	for _, runs := range shards {
		for _, r := range runs {
			if r.run != "" || r.skip != "" {
				split[r.packages[0]] = true
			}
		}
	}
	for p := range split {
		count[p]++
	}
	return count
}

func TestLoadTimingsMergesTheTargetsShards(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("linux-amd64-go-1.json", `{"target":"linux-amd64","packages":{"p":{"seconds":3,"tests":{"TestA":2}}}}`)
	write("linux-amd64-go-2.json", `{"target":"linux-amd64","packages":{"p":{"seconds":4,"tests":{"TestB":3}},"q":{"seconds":1}}}`)
	write("windows-amd64-go-1.json", `{"target":"windows-amd64","packages":{"p":{"seconds":90}}}`)
	got, err := loadTimings(dir, "linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]packageTimings{"p": {Seconds: 7, Tests: map[string]float64{"TestA": 2, "TestB": 3}}, "q": {Seconds: 1}}
	if fmt.Sprint(got.Packages) != fmt.Sprint(want) {
		t.Errorf("timings = %v, want %v", got.Packages, want)
	}
	if none, err := loadTimings(filepath.Join(dir, "missing"), "linux-amd64"); err != nil || len(none.Packages) != 0 {
		t.Errorf("timings of a missing dir = %v, %v, want none", none, err)
	}
}

func TestLoadTimingsFillsAMissingPackageFromAnotherTarget(t *testing.T) {
	dir := t.TempDir()
	writeTimings(t, dir, "linux-amd64-e2e-1.json", `{"target":"linux-amd64","cpus":4,"packages":{"e2e":{"seconds":866},"p":{"seconds":5}}}`)
	writeTimings(t, dir, "windows-amd64-e2e-1.json", `{"target":"windows-amd64","cpus":2,"packages":{"e2e":{"seconds":999},"p":{"seconds":77},"q":{"seconds":3}}}`)
	writeTimings(t, dir, "darwin-arm64-e2e-1.json", `{"target":"darwin-arm64","packages":{"p":{"seconds":2}}}`)
	got, err := loadTimings(dir, "darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]packageTimings{"e2e": {Seconds: 866}, "p": {Seconds: 2}, "q": {Seconds: 3}}
	if fmt.Sprint(got.Packages) != fmt.Sprint(want) {
		t.Errorf("darwin-arm64 timings = %v, want its own p, linux-amd64's e2e and windows-amd64's q: %v", got.Packages, want)
	}
	if got.CPUs != 4 {
		t.Errorf("darwin-arm64 CPUs = %d, want linux-amd64's 4", got.CPUs)
	}
}

func TestPartitionSpreadsHeavyPackagesOnATargetWithoutTimings(t *testing.T) {
	dir := t.TempDir()
	writeTimings(t, dir, "linux-amd64-e2e-1.json", `{"target":"linux-amd64","cpus":4,"packages":{"e2e":{"seconds":866},"fakes/tracker":{"seconds":338},"fakes/world/sim":{"seconds":650},"a":{"seconds":5},"b":{"seconds":5},"c":{"seconds":5},"d":{"seconds":5}}}`)
	past, err := loadTimings(dir, "darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	shards := partition(packages("a", "b", "c", "d", "e2e", "fakes/tracker", "fakes/world/sim"), past, 2, true)
	shardOf := map[string]int{}
	for i, runs := range shards {
		for _, r := range runs {
			for _, p := range r.packages {
				shardOf[p] = i + 1
			}
		}
	}
	if shardOf["e2e"] == shardOf["fakes/world/sim"] {
		t.Errorf("e2e and fakes/world/sim both run on shard %d of %v, want them apart", shardOf["e2e"], shards)
	}
}

func writeTimings(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestGoTestLogPrintsWhatGoTestPrintsWithoutV(t *testing.T) {
	events := []string{
		`{"Action":"start","Package":"ok/pkg"}`,
		`{"Action":"run","Package":"ok/pkg","Test":"TestA"}`,
		`{"Action":"output","Package":"ok/pkg","Test":"TestA","Output":"=== RUN   TestA\n"}`,
		`{"Action":"output","Package":"ok/pkg","Test":"TestA","Output":"    a_test.go:1: chatter\n"}`,
		`{"Action":"pass","Package":"ok/pkg","Test":"TestA","Elapsed":1.5}`,
		`{"Action":"output","Package":"ok/pkg","Output":"PASS\n"}`,
		`{"Action":"output","Package":"ok/pkg","Output":"ok  \tok/pkg\t1.6s\n"}`,
		`{"Action":"pass","Package":"ok/pkg","Elapsed":1.6}`,
		`{"ImportPath":"bad/pkg [bad/pkg.test]","Action":"build-output","Output":"bad.go:1: syntax error\n"}`,
		`{"Action":"run","Package":"fail/pkg","Test":"TestB"}`,
		`{"Action":"output","Package":"fail/pkg","Test":"TestB","Output":"=== RUN   TestB\n"}`,
		`{"Action":"run","Package":"fail/pkg","Test":"TestB/sub"}`,
		`{"Action":"output","Package":"fail/pkg","Test":"TestB/sub","Output":"    b_test.go:9: boom\n"}`,
		`{"Action":"output","Package":"fail/pkg","Test":"TestB/sub","Output":"    --- FAIL: TestB/sub (0.10s)\n"}`,
		`{"Action":"fail","Package":"fail/pkg","Test":"TestB/sub","Elapsed":0.1}`,
		`{"Action":"output","Package":"fail/pkg","Test":"TestB","Output":"--- FAIL: TestB (0.20s)\n"}`,
		`{"Action":"fail","Package":"fail/pkg","Test":"TestB","Elapsed":0.2}`,
		`{"Action":"output","Package":"fail/pkg","Output":"FAIL\n"}`,
		`{"Action":"output","Package":"fail/pkg","Output":"FAIL\tfail/pkg\t0.3s\n"}`,
		`{"Action":"fail","Package":"fail/pkg","Elapsed":0.3}`,
		`not json from go test`,
	}
	var b strings.Builder
	log := newGoTestLog(&b)
	for _, e := range events {
		if _, err := log.Write([]byte(e + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := log.flush(); err != nil {
		t.Fatal(err)
	}
	want := "ok  \tok/pkg\t1.6s\n" +
		"bad.go:1: syntax error\n" +
		"    b_test.go:9: boom\n    --- FAIL: TestB/sub (0.10s)\n" +
		"--- FAIL: TestB (0.20s)\n" +
		"FAIL\nFAIL\tfail/pkg\t0.3s\n" +
		"not json from go test\n"
	if b.String() != want {
		t.Errorf("printed\n%s\nwant\n%s", b.String(), want)
	}
	wantTimings := map[string]packageTimings{
		"ok/pkg":   {Seconds: 1.6, Tests: map[string]float64{"TestA": 1.5}},
		"fail/pkg": {Seconds: 0.3, Tests: map[string]float64{"TestB": 0.2}},
	}
	if fmt.Sprint(log.timings.Packages) != fmt.Sprint(wantTimings) {
		t.Errorf("timings = %v, want %v", log.timings.Packages, wantTimings)
	}
}

func TestLoadTimingsFillsAnUnseenTestFromAnotherTargetAtThisTargetsScale(t *testing.T) {
	dir := t.TempDir()
	writeTimings(t, dir, "linux-amd64-browser-1.json", `{"target":"linux-amd64","packages":{"browser":{"seconds":40,"tests":{"[webkit-phone] › a":10,"[webkit-phone] › slow":30}}}}`)
	writeTimings(t, dir, "darwin-arm64-browser-2.json", `{"target":"darwin-arm64","packages":{"browser":{"seconds":20,"tests":{"[webkit-phone] › a":20}}}}`)
	got, err := loadTimings(dir, "darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"[webkit-phone] › a": 20, "[webkit-phone] › slow": 60}
	if fmt.Sprint(got.Packages["browser"].Tests) != fmt.Sprint(want) {
		t.Errorf("darwin-arm64 browser tests = %v, want its own a and linux-amd64's slow at twice its seconds: %v", got.Packages["browser"].Tests, want)
	}
}
