package main

import (
	"cmp"
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
	register("daily-issues", "open or update the daily check's issues: one that names its failed jobs and the commits since the last full-matrix run that passed, one for each service whose online conformance tests failed, one that proposes ci/quarantine.tsv rows for tests labeled flaky twice in 7 days and lists the quarantined tests that passed for 14, and one for expired rows, which fail the command", runDailyIssues)
}

// fullMatrixEvery is how old the last full-matrix daily check that passed
// may get before the next one tests the full matrix again on an unchanged
// tree, since the runner images and toolchains change under it.
const fullMatrixEvery = 7 * 24 * time.Hour

// The daily check files its own issues for these jobs' failures. A
// quarantined test's failure fails no check.
const (
	onlineJob     = "Online conformance tests"
	linkJob       = "Link throughput test"
	quarantineJob = "Quarantined tests"
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
	return quarantineIssues(ctx, repo, runID, failures, url)
}

// quarantineIssues proposes quarantine rows for the tests labeled flaky
// twice or more in the last 7 days, this run's failures among them, lists
// the quarantined tests that passed in each of the last 14 days' daily
// checks, and fails on an expired row.
func quarantineIssues(ctx context.Context, repo string, runID int64, failures []failure, url string) error {
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	list, err := loadQuarantine(root)
	if err != nil {
		return err
	}
	today := day(time.Now())
	week, err := ghList[ghRun](ctx, fmt.Sprintf("repos/%s/actions/runs?created=%%3E%%3D%s&per_page=100", repo, today.AddDate(0, 0, -7).Format(time.DateOnly)), ".workflow_runs", true)
	if err != nil {
		return err
	}
	week = slices.DeleteFunc(week, func(r ghRun) bool {
		return r.ID == runID || r.Conclusion != "failure" || !slices.Contains(statsWorkflows, r.Path)
	})
	past, _, err := downloadFailures(ctx, repo, week)
	if err != nil {
		return err
	}
	dailies, err := ghList[ghRun](ctx, fmt.Sprintf("repos/%s/actions/workflows/daily.yml/runs?branch=main&created=%%3E%%3D%s&per_page=100", repo, today.AddDate(0, 0, -quarantineDays).Format(time.DateOnly)), ".workflow_runs", true)
	if err != nil {
		return err
	}
	byDay, err := downloadQuarantineResults(ctx, repo, slices.DeleteFunc(dailies, func(r ghRun) bool { return r.ID == runID }))
	if err != nil {
		return err
	}
	proposals := proposeQuarantine(list, slices.Concat(failures, past), today)
	removals := readyToRemove(list, byDay, today)
	if len(proposals) > 0 || len(removals) > 0 {
		about := "The daily check comments here with the flaky tests to add to " + quarantinePath + " and the quarantined tests to remove from it."
		comment := func(number int) string { return quarantineComment(proposals, removals, number, url) }
		if err := commentOnIssue(ctx, repo, "Flaky tests to quarantine", about, comment); err != nil {
			return err
		}
	}
	expired := expiredRows(list, today)
	if len(expired) == 0 {
		return nil
	}
	if err := openOrUpdateIssue(ctx, repo, expiredIssue(expired, url)); err != nil {
		return err
	}
	return expiredError(expired)
}

func day(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// downloadQuarantineResults reads the quarantine-* artifacts of each run,
// by the day the run started.
func downloadQuarantineResults(ctx context.Context, repo string, runs []ghRun) (map[string][]quarantineResult, error) {
	dir, err := os.MkdirTemp("", "quarantine-")
	if err != nil {
		return nil, fmt.Errorf("make a dir for the quarantine artifacts: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	byDay := map[string][]quarantineResult{}
	for _, r := range runs {
		runDir := filepath.Join(dir, strconv.FormatInt(r.ID, 10))
		n, err := downloadArtifacts(ctx, repo, r.ID, "quarantine-", runDir)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		got, err := readQuarantineResults(runDir)
		if err != nil {
			return nil, err
		}
		on := day(r.CreatedAt).Format(time.DateOnly)
		byDay[on] = append(byDay[on], got...)
	}
	return byDay, nil
}

func readQuarantineResults(dir string) ([]quarantineResult, error) {
	var all []quarantineResult
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".json" {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		var got []quarantineResult
		if err := json.Unmarshal(b, &got); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		all = append(all, got...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read the quarantined tests' results: %w", err)
	}
	return all, nil
}

// quarantineProposal is a row the daily check proposes for a test labeled
// flaky in its tier at least twice. A proposal for a test the list has
// already replaces that row with one that adds the targets it was flaky
// on.
type quarantineProposal struct {
	row      quarantined
	flaky    int
	replaces bool
}

func proposeQuarantine(list []quarantined, failures []failure, today time.Time) []quarantineProposal {
	type key struct{ tier, test string }
	count := map[key]int{}
	targets := map[key][]string{}
	for _, f := range failures {
		k := key{f.Tier, testID(f.Package, f.Test)}
		if _, ok := tierRunners[f.Tier]; !ok || f.Rerun != flaky || k.test == "" {
			continue
		}
		count[k]++
		if !slices.Contains(targets[k], f.Target) {
			targets[k] = append(targets[k], f.Target)
		}
	}
	var out []quarantineProposal
	for _, k := range slices.SortedFunc(maps.Keys(count), func(a, b key) int { return cmp.Or(cmp.Compare(a.tier, b.tier), cmp.Compare(a.test, b.test)) }) {
		if count[k] < 2 {
			continue
		}
		p := quarantineProposal{row: quarantined{tier: k.tier, test: k.test, added: today, expires: today.AddDate(0, 0, quarantineDays)}, flaky: count[k]}
		want := targets[k]
		if i := slices.IndexFunc(list, func(q quarantined) bool { return q.tier == k.tier && q.test == k.test }); i >= 0 {
			listed := list[i]
			if !slices.ContainsFunc(want, func(t string) bool { return !listed.on(t) }) {
				continue
			}
			p.replaces = true
			want = append(slices.Clone(listed.targets), want...)
		}
		p.row.targets = slices.Compact(slices.Sorted(slices.Values(want)))
		out = append(out, p)
	}
	return out
}

// readyToRemove are the rows whose test passed in the daily checks on
// each of the quarantineDays days before today, by the day of each run.
func readyToRemove(list []quarantined, byDay map[string][]quarantineResult, today time.Time) []quarantined {
	var ready []quarantined
	for _, q := range list {
		passed := true
		for d := 1; d <= quarantineDays && passed; d++ {
			var ran bool
			for _, r := range byDay[today.AddDate(0, 0, -d).Format(time.DateOnly)] {
				if r.Tier == q.tier && r.Test == q.test {
					ran = true
					passed = passed && r.Result == "passed"
				}
			}
			passed = passed && ran
		}
		if passed {
			ready = append(ready, q)
		}
	}
	return ready
}

// expiredRows are the rows whose expiry day has come.
func expiredRows(list []quarantined, today time.Time) []quarantined {
	return slices.DeleteFunc(slices.Clone(list), func(q quarantined) bool { return today.Before(q.expires) })
}

func quarantineComment(proposals []quarantineProposal, removals []quarantined, number int, run string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From the daily check: %s\n", run)
	if len(proposals) > 0 {
		fmt.Fprintf(&b, "\n### Flaky tests to quarantine\n\nThese tests were labeled flaky twice or more in the last 7 days. To quarantine them, add these rows to %s. Each row names this issue and expires in %d days.\n\n| Test | Tier | Flaky on | Times flaky | Row |\n|---|---|---|---|---|\n", quarantinePath, quarantineDays)
		var rows []string
		for _, p := range proposals {
			p.row.issue = number
			rows = append(rows, p.row.row())
			action := "new"
			if p.replaces {
				action = "replaces the listed row"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %d | %s |\n", markdownCell(p.row.test), p.row.tier, strings.Join(p.row.targets, ", "), p.flaky, action)
		}
		fmt.Fprintf(&b, "\n```tsv\n%s\n```\n", strings.Join(rows, "\n"))
	}
	if len(removals) > 0 {
		fmt.Fprintf(&b, "\n### Quarantined tests to remove\n\nThese quarantined tests passed in the daily check on each of the last %d days. Remove their rows from %s:\n\n", quarantineDays, quarantinePath)
		for _, q := range removals {
			fmt.Fprintf(&b, "- %s in the %s tier, for #%d\n", markdownCell(q.test), q.tier, q.issue)
		}
	}
	return b.String()
}

func expiredIssue(expired []quarantined, run string) issue {
	var b strings.Builder
	fmt.Fprintf(&b, "The daily check fails while these rows of %s have expired: %s\n\n", quarantinePath, run)
	for _, q := range expired {
		fmt.Fprintf(&b, "- %s in the %s tier, quarantined for #%d, expired on %s.\n", markdownCell(q.test), q.tier, q.issue, q.expires.Format(time.DateOnly))
	}
	b.WriteString("\nFix each test and remove its row, or delete the test.\n")
	return issue{"Quarantine entries expired", b.String()}
}

func expiredError(expired []quarantined) error {
	var names []string
	for _, q := range expired {
		names = append(names, fmt.Sprintf("%s in the %s tier (#%d, expired on %s)", q.test, q.tier, q.issue, q.expires.Format(time.DateOnly)))
	}
	return fmt.Errorf("%d rows of %s have expired: %s. Fix each test and remove its row, or delete the test", len(expired), quarantinePath, strings.Join(names, ", "))
}

type issue struct{ title, body string }

// dailyIssues are the issues a failed daily check opens or updates: one
// for each service whose online conformance tests failed, and one naming
// every other failed job, with the commits since the last full-matrix run
// that passed. The link throughput test files its own, and the quarantine
// lane none.
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
		if j.Conclusion != "failure" || lane == linkJob || lane == quarantineJob || lane == onlineJob && len(services) > 0 {
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
	n, err := findOpenIssue(ctx, repo, is.title)
	if err != nil {
		return err
	}
	if n > 0 {
		fmt.Printf("Commenting on #%d, %s.\n", n, is.title)
		return run(ctx, "", nil, "gh", "issue", "comment", strconv.Itoa(n), "--repo", repo, "--body", is.body)
	}
	fmt.Printf("Opening an issue, %s.\n", is.title)
	return run(ctx, "", nil, "gh", "issue", "create", "--repo", repo, "--title", is.title, "--body", is.body)
}

// commentOnIssue comments on the open issue titled title, which it opens
// with about as its text when none is, since the comment names the issue's
// number.
func commentOnIssue(ctx context.Context, repo, title, about string, comment func(number int) string) error {
	n, err := findOpenIssue(ctx, repo, title)
	if err != nil {
		return err
	}
	if n == 0 {
		fmt.Printf("Opening an issue, %s.\n", title)
		create := exec.CommandContext(ctx, "gh", "issue", "create", "--repo", repo, "--title", title, "--body", about)
		create.Stderr = os.Stderr
		out, err := create.Output()
		if err != nil {
			return fmt.Errorf("open the issue %q: %w", title, err)
		}
		u := strings.TrimSpace(string(out))
		if n, err = strconv.Atoi(path.Base(u)); err != nil {
			return fmt.Errorf("open the issue %q: gh printed %q, which doesn't end in its number", title, u)
		}
	}
	fmt.Printf("Commenting on #%d, %s.\n", n, title)
	return run(ctx, "", nil, "gh", "issue", "comment", strconv.Itoa(n), "--repo", repo, "--body", comment(n))
}

// findOpenIssue is the number of the open issue titled title, 0 when none
// is.
func findOpenIssue(ctx context.Context, repo, title string) (int, error) {
	list := exec.CommandContext(ctx, "gh", "issue", "list", "--repo", repo, "--state", "open", "--search", strconv.Quote(title)+" in:title", "--json", "number,title")
	list.Stderr = os.Stderr
	out, err := list.Output()
	if err != nil {
		return 0, fmt.Errorf("find the issue %q: %w", title, err)
	}
	var open []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(out, &open); err != nil {
		return 0, fmt.Errorf("find the issue %q: %w", title, err)
	}
	for _, o := range open {
		if o.Title == title {
			return o.Number, nil
		}
	}
	return 0, nil
}
