package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func init() {
	register("daily", "choose what the daily check runs: kind daily, the online conformance tests alone, or kind schedule, the full matrix too, when no daily check on main has passed the full matrix on this tree in the last 7 days; also sets the since output to the commit of the last one that passed", runDaily)
	register("daily-issues", "open or update the issues of a failed daily check: one that names its failed jobs and the commits since the last full-matrix run that passed, and one for each service whose online conformance tests failed", runDailyIssues)
}

// fullMatrixEvery is how old the last full-matrix daily check that passed
// may get before the next one tests the full matrix again on an unchanged
// tree, since the runner images and toolchains change under it.
const fullMatrixEvery = 7 * 24 * time.Hour

// The daily check files its own issues for these jobs' failures.
const (
	onlineJob = "Online conformance tests"
	linkJob   = "Link throughput test"
)

func runDaily(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("daily takes no arguments, not %q", args)
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	tree, err := gitOutput(ctx, ".", nil, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return err
	}
	green, err := ghList[ghRun](ctx, "repos/"+repo+"/actions/workflows/daily.yml/runs?branch=main&status=success&per_page=30", ".workflow_runs", false)
	if err != nil {
		return err
	}
	ranFull := func(r ghRun) (bool, error) {
		jobs, err := runJobs(ctx, repo, r.ID, "latest")
		return ranFullMatrix(jobs), err
	}
	d, err := decideDaily(green, ranFull, strings.TrimSpace(string(tree)), time.Now())
	if err != nil {
		return err
	}
	fmt.Println(d.reason)
	if err := githubOutput("kind", d.kind); err != nil {
		return err
	}
	return githubOutput("since", d.since)
}

// ranFullMatrix says whether a daily check's jobs tested the full matrix,
// which builds the binaries where the online tests alone build none.
func ranFullMatrix(jobs []ghJob) bool {
	return slices.ContainsFunc(jobs, func(j ghJob) bool {
		lane, _ := jobLane(j.Name)
		return lane == "Build" && j.Conclusion == "success"
	})
}

// daily is what a daily check runs: kind is the plan's run kind, and since
// the commit of the last full-matrix daily check that passed, empty when
// none has.
type daily struct{ kind, since, reason string }

// decideDaily picks the daily check's kind from the daily checks on main
// that passed, newest first. It tests the full matrix unless the newest
// one that ran it tested this tree less than fullMatrixEvery ago.
func decideDaily(green []ghRun, ranFull func(ghRun) (bool, error), tree string, now time.Time) (daily, error) {
	for _, r := range green {
		full, err := ranFull(r)
		if err != nil {
			return daily{}, err
		}
		if !full {
			continue
		}
		d := daily{kind: "schedule", since: r.HeadSHA}
		on := r.CreatedAt.Format("Jan 2")
		switch {
		case r.HeadCommit.TreeID != tree:
			d.reason = fmt.Sprintf("Run %d passed the full matrix on %s, on an older tree. This run tests the full matrix.", r.ID, on)
		case now.Sub(r.CreatedAt) >= fullMatrixEvery:
			d.reason = fmt.Sprintf("Run %d passed the full matrix on this tree on %s, 7 or more days ago. This run tests the full matrix again.", r.ID, on)
		default:
			d.kind = "daily"
			d.reason = fmt.Sprintf("Run %d passed the full matrix on this tree on %s. Only the online conformance tests run in this check.", r.ID, on)
		}
		return d, nil
	}
	return daily{kind: "schedule", reason: "No daily check on main has passed the full matrix yet. This run tests the full matrix."}, nil
}

func runDailyIssues(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("daily-issues", flag.ContinueOnError)
	dir := flags.String("failures", "", "directory of the run's failures-* artifacts")
	since := flags.String("since", "", "commit of the last full-matrix daily check that passed, as the daily command's since output")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return fmt.Errorf("read GITHUB_RUN_ID: %w", err)
	}
	jobs, err := runJobs(ctx, repo, runID, "latest")
	if err != nil {
		return err
	}
	failures, err := readFailures(*dir)
	if err != nil {
		return err
	}
	var commits []string
	if *since != "" {
		out, err := gitOutput(ctx, ".", nil, "log", "--first-parent", "--format=%h %s", *since+"..HEAD")
		if err != nil {
			return err
		}
		commits = strings.Split(strings.TrimSpace(string(out)), "\n")
		commits = slices.DeleteFunc(commits, func(c string) bool { return c == "" })
	}
	url := fmt.Sprintf("%s/%s/actions/runs/%d", os.Getenv("GITHUB_SERVER_URL"), repo, runID)
	for _, is := range dailyIssues(jobs, failures, commits, *since, url) {
		if err := openOrUpdateIssue(ctx, repo, is); err != nil {
			return err
		}
	}
	return nil
}

type issue struct{ title, body string }

// dailyIssues are the issues a failed daily check opens or updates: one
// for each service whose online conformance tests failed, and one naming
// every other failed job, with the commits since the last full-matrix run
// that passed. The link throughput test files its own.
func dailyIssues(jobs []ghJob, failures []failure, commits []string, since, run string) []issue {
	services := map[string][]failure{}
	for _, f := range failures {
		if f.Tier == "online" {
			s := path.Base(f.Package)
			services[s] = append(services[s], f)
		}
	}
	var failed []string
	for _, j := range jobs {
		lane, _ := jobLane(j.Name)
		if j.Conclusion != "failure" || lane == linkJob || lane == onlineJob && len(services) > 0 {
			continue
		}
		failed = append(failed, jobTitle(j.Name))
	}
	var issues []issue
	if len(failed) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "The daily check failed: %s\n\nThese jobs failed:\n\n", run)
		for _, name := range failed {
			fmt.Fprintf(&b, "- %s\n", name)
		}
		switch {
		case since == "":
			b.WriteString("\nNo daily check on main has passed the full matrix yet.\n")
		case len(commits) == 0:
			fmt.Fprintf(&b, "\nNo commits merged since %s, the last commit whose full-matrix run passed.\n", short(since))
		default:
			fmt.Fprintf(&b, "\nThese commits merged since %s, the last commit whose full-matrix run passed:\n\n", short(since))
			for _, c := range commits {
				fmt.Fprintf(&b, "- %s\n", c)
			}
		}
		issues = append(issues, issue{"Daily check failed", b.String()})
	}
	for _, s := range slices.Sorted(maps.Keys(services)) {
		var b strings.Builder
		fmt.Fprintf(&b, "The %s conformance tests failed against the real service in the daily check: %s\n\n", s, run)
		for _, f := range services[s] {
			fmt.Fprintf(&b, "- %s\n", markdownCell(strings.TrimSpace(f.Test+" "+f.Message)))
		}
		issues = append(issues, issue{"Online conformance tests failed for " + s, b.String()})
	}
	return issues
}

func short(sha string) string { return sha[:min(len(sha), 12)] }

// readFailures reads the failures.json files under dir. A missing dir
// holds none, as when no job uploaded any.
func readFailures(dir string) ([]failure, error) {
	var all []failure
	if dir == "" {
		return all, nil
	}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".json" {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read the failed tests: %w", err)
		}
		var got []failure
		if err := json.Unmarshal(b, &got); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		all = append(all, got...)
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return all, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the failed tests: %w", err)
	}
	return all, nil
}

// openOrUpdateIssue comments on the open issue titled is.title, or opens
// one when none is.
func openOrUpdateIssue(ctx context.Context, repo string, is issue) error {
	list := exec.CommandContext(ctx, "gh", "issue", "list", "--repo", repo, "--state", "open", "--search", strconv.Quote(is.title)+" in:title", "--json", "number,title")
	list.Stderr = os.Stderr
	out, err := list.Output()
	if err != nil {
		return fmt.Errorf("find the issue %q: %w", is.title, err)
	}
	var open []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(out, &open); err != nil {
		return fmt.Errorf("find the issue %q: %w", is.title, err)
	}
	for _, o := range open {
		if o.Title == is.title {
			fmt.Printf("Commenting on #%d, %s.\n", o.Number, is.title)
			return run(ctx, "", nil, "gh", "issue", "comment", strconv.Itoa(o.Number), "--repo", repo, "--body", is.body)
		}
	}
	fmt.Printf("Opening an issue, %s.\n", is.title)
	return run(ctx, "", nil, "gh", "issue", "create", "--repo", repo, "--title", is.title, "--body", is.body)
}
