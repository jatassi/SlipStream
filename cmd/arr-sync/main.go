// Command arr-sync brings Sonarr's, Radarr's and TRaSH Guides' parser
// knowledge into SlipStream at the commits cmd/arr-sync/pins.json names.
// It is a maintainer tool and never ships in the release binary. Run it
// as make arr-sync. It needs the network only to fetch a commit it hasn't
// fetched before, and CI never runs it.
//
//	arr-sync [--repo DIR] [--checkouts DIR]
//
// It fetches each pinned commit as a shallow, blobless, sparse git
// checkout under --checkouts, then replaces what an earlier run wrote with
// these files.
//
//   - Each owner's internal/upstream package holds every upstream regular
//     expression as data, keyed by its upstream name.
//   - internal/formats/zz_dialects.go holds the enums the format dialects map.
//   - Each owner's testdata/upstream holds every [TestCase] row as JSON.
//   - internal/naming/testdata/arr holds Sonarr's and Radarr's file naming
//     [TestCase] rows as naming cases.
//   - The TRaSH snapshots, the fixture titles and the fuzz seeds.
//   - Each owner's upstream.json, and the notices of the code SlipStream
//     ships from upstream.
//
// It prints the review list: each upstream file that changed since the
// last run, with the hand-ported Go files that port it. A rerun at the
// same pins changes nothing.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "arr-sync:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("arr-sync", flag.ContinueOnError)
	repo := fs.String("repo", ".", "the SlipStream repository")
	checkouts := fs.String("checkouts", "", "where the upstream checkouts live. It defaults to the user cache directory.")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("arr-sync: %w", err)
	}
	root, err := os.OpenRoot(*repo)
	if err != nil {
		return fmt.Errorf("open the repository: %w", err)
	}
	defer func() { _ = root.Close() }()
	p, err := readPins(root)
	if err != nil {
		return err
	}
	if *checkouts == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("find the cache directory: %w", err)
		}
		*checkouts = filepath.Join(cache, "slipstream", "arr-sync")
	}
	cos, err := fetch(ctx, *checkouts, p)
	if err != nil {
		return err
	}
	t := &tree{files: map[string][]byte{}}
	patterns, err := patternTables(t, p, cos)
	if err != nil {
		return err
	}
	rows, titles, err := fixtureFiles(t, p, cos)
	if err != nil {
		return err
	}
	if err := dialectTable(t, p, cos); err != nil {
		return err
	}
	formats, err := trashSnapshot(t, p, cos[p.Trash.From])
	if err != nil {
		return err
	}
	conditions, err := trashReaders(t, p, cos[p.Trash.From])
	if err != nil {
		return err
	}
	namings, err := trashNaming(t, p, cos[p.Trash.From])
	if err != nil {
		return err
	}
	cases, err := organizerCases(t, p, cos)
	if err != nil {
		return err
	}
	multiline := titlesFile(t, p, titles)
	seeds, err := fuzzSeeds(t, p, cos)
	if err != nil {
		return err
	}
	if err := noticeFiles(t, cos); err != nil {
		return err
	}
	review, err := provenanceFiles(t, root, p, cos)
	if err != nil {
		return err
	}
	changed, err := t.write(root)
	if err != nil {
		return err
	}
	w := &report{out: out}
	w.section("Patterns per owner", patterns)
	w.section("Rows per fixture file", rows)
	w.section("Fuzz seeds per file", seeds)
	w.section("Naming cases per file", cases)
	w.line("TRaSH formats in the snapshot: %d. Stream reader conditions: %d. Naming files: %d.", formats, conditions, namings)
	w.line("Fixture titles: %d, of which %d span lines and are left out of %s.", len(titles), multiline, p.Trash.Titles)
	w.list("Review these upstream changes", review)
	w.list("Files changed", changed)
	return w.err
}

type report struct {
	out io.Writer
	err error
}

func (r *report) line(format string, args ...any) {
	if r.err == nil {
		_, r.err = fmt.Fprintf(r.out, format+"\n", args...)
	}
}

func (r *report) section(title string, counts map[string]int) {
	r.line("%s:", title)
	for _, k := range slices.Sorted(maps.Keys(counts)) {
		r.line("  %s: %d", k, counts[k])
	}
}

func (r *report) list(title string, lines []string) {
	if len(lines) == 0 {
		r.line("%s: none.", title)
		return
	}
	r.line("%s:", title)
	for _, l := range lines {
		r.line("  %s", l)
	}
}
