package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func init() {
	register("ci-stats", "print a Markdown report of CI health from the last -days of Actions runs, read through gh with GH_TOKEN: wave wall time, the merge check's first-attempt pass rate, failure and flaky counts per lane and target, the daily check's streak and queue time per runner OS", runCIStats)
}

// statsWorkflows are the workflows ci-stats reads.
var statsWorkflows = []string{".github/workflows/pr.yml", ".github/workflows/push.yml", ".github/workflows/daily.yml"}

func runCIStats(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("ci-stats", flag.ContinueOnError)
	days := flags.Int("days", 7, "how many days of runs to read")
	repo := flags.String("repo", os.Getenv("GITHUB_REPOSITORY"), "the repository, as owner/name")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if *days < 1 || *repo == "" {
		return fmt.Errorf("ci-stats needs -days of 1 or more and -repo, not %d and %q", *days, *repo)
	}
	from := time.Now().UTC().AddDate(0, 0, -*days)
	runs, err := ghList[ghRun](ctx, fmt.Sprintf("repos/%s/actions/runs?created=%%3E%%3D%s&per_page=100", *repo, from.Format(time.DateOnly)), ".workflow_runs", true)
	if err != nil {
		return err
	}
	runs = slices.DeleteFunc(runs, func(r ghRun) bool { return !slices.Contains(statsWorkflows, r.Path) })
	jobs := map[int64][]ghJob{}
	for _, r := range runs {
		if jobs[r.ID], err = runJobs(ctx, *repo, r.ID, "all"); err != nil {
			return err
		}
	}
	daily, err := ghList[ghRun](ctx, "repos/"+*repo+"/actions/workflows/daily.yml/runs?branch=main&per_page=100", ".workflow_runs", false)
	if err != nil {
		return err
	}
	failed := slices.DeleteFunc(slices.Clone(runs), func(r ghRun) bool {
		return !slices.ContainsFunc(jobs[r.ID], func(j ghJob) bool { return j.Conclusion == "failure" })
	})
	failures, artifacts, err := downloadFailures(ctx, *repo, failed)
	if err != nil {
		return err
	}
	s := ciStats(runs, jobs, daily, failures, artifacts)
	return s.write(os.Stdout, *days)
}

// downloadFailures reads the failures-* artifacts of each run, and counts
// the artifacts it read.
func downloadFailures(ctx context.Context, repo string, runs []ghRun) ([]failure, int, error) {
	dir, err := os.MkdirTemp("", "failures-")
	if err != nil {
		return nil, 0, fmt.Errorf("make a dir for the failures artifacts: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	var all []failure
	read := 0
	for _, r := range runs {
		runDir := filepath.Join(dir, strconv.FormatInt(r.ID, 10))
		n, err := downloadArtifacts(ctx, repo, r.ID, "failures-", runDir)
		if err != nil {
			return nil, 0, err
		}
		f, err := readFailures(runDir)
		if err != nil {
			return nil, 0, err
		}
		all = append(all, f...)
		read += n
	}
	return all, read, nil
}

// downloadArtifacts downloads into dir the run's unexpired artifacts whose
// names start with prefix, and counts them.
func downloadArtifacts(ctx context.Context, repo string, runID int64, prefix, dir string) (int, error) {
	artifacts, err := ghList[struct {
		Name    string `json:"name"`
		Expired bool   `json:"expired"`
	}](ctx, fmt.Sprintf("repos/%s/actions/runs/%d/artifacts?per_page=100", repo, runID), ".artifacts", true)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, a := range artifacts {
		if strings.HasPrefix(a.Name, prefix) && !a.Expired {
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	id := strconv.FormatInt(runID, 10)
	return n, run(ctx, "", nil, "gh", "run", "download", id, "--repo", repo, "--pattern", prefix+"*", "--dir", dir)
}

type laneCount struct{ jobs, failed int }

// stats are the CI health measures of a set of runs.
type stats struct {
	// waveGreen and waveRed are the wall minutes of the wave checks' first
	// attempts that passed and that failed.
	waveGreen, waveRed []float64
	// mergeGreen of mergeRuns merge checks passed on their first attempt.
	mergeGreen, mergeRuns int
	lanes                 map[laneTarget]laneCount
	flaky                 map[laneTarget]int
	// artifacts is how many failures artifacts the flaky counts come from.
	// With none they come from jobs that passed when the run ran again.
	artifacts int
	// streak is how many daily checks in a row ended as the newest did.
	streak           int
	streakConclusion string
	// queue holds each started job's minutes in the queue, by runner OS.
	queue map[string][]float64
}

// gates fail whenever a lane does, so they count as no lane.
var gates = []string{"All required checks passed", "All push checks passed"}

// ciStats measures runs, whose jobs of every attempt jobs holds, and the
// daily checks on main, newest first. failures are the tests that the
// failures artifacts name, from that many artifacts.
func ciStats(runs []ghRun, jobs map[int64][]ghJob, daily []ghRun, failures []failure, artifacts int) stats {
	s := stats{lanes: map[laneTarget]laneCount{}, flaky: map[laneTarget]int{}, artifacts: artifacts, queue: map[string][]float64{}}
	for _, r := range runs {
		if r.Path == ".github/workflows/pr.yml" {
			s.addPullRequest(r, jobs[r.ID])
		}
		s.addJobs(jobs[r.ID])
		if artifacts == 0 {
			for _, lt := range rerunPassed(jobs[r.ID]) {
				s.flaky[lt]++
			}
		}
	}
	for _, f := range failures {
		if f.Rerun == flaky {
			s.flaky[laneTarget{f.Tier, f.Target}]++
		}
	}
	for _, r := range daily {
		if r.Conclusion != "success" && r.Conclusion != "failure" {
			continue
		}
		if s.streak > 0 && r.Conclusion != s.streakConclusion {
			break
		}
		s.streakConclusion = r.Conclusion
		s.streak++
	}
	return s
}

// addPullRequest counts a wave or merge check by how its first attempt
// ended.
func (s *stats) addPullRequest(r ghRun, jobs []ghJob) {
	outcome, wall := firstAttempt(r, jobs)
	wave := strings.HasPrefix(r.base(), "feature/")
	switch {
	case wave && outcome == "success":
		s.waveGreen = append(s.waveGreen, wall)
	case wave && outcome == "failure":
		s.waveRed = append(s.waveRed, wall)
	case r.base() == "main" && (outcome == "success" || outcome == "failure"):
		s.mergeRuns++
		if outcome == "success" {
			s.mergeGreen++
		}
	}
}

// addJobs counts each job that started into its runner OS's queue times,
// and each that passed or failed into its lane on its target.
func (s *stats) addJobs(jobs []ghJob) {
	for _, j := range jobs {
		if !j.StartedAt.IsZero() && j.Conclusion != "skipped" {
			o := runnerOS(j.Labels)
			s.queue[o] = append(s.queue[o], j.StartedAt.Sub(j.CreatedAt).Minutes())
		}
		lane, target := jobLane(j.Name)
		if slices.Contains(gates, lane) || j.Conclusion != "success" && j.Conclusion != "failure" {
			continue
		}
		c := s.lanes[laneTarget{lane, target}]
		c.jobs++
		if j.Conclusion == "failure" {
			c.failed++
		}
		s.lanes[laneTarget{lane, target}] = c
	}
}

// canceled is the Actions API's conclusion of a canceled job.
const canceled = "cancelled" //nolint:misspell // the Actions API's spelling

// firstAttempt is how a run's first attempt ended, and its wall minutes
// from the run's creation to its last job's end. It failed when a job
// failed, was canceled when a job was and none failed, and passed when a
// job passed and none failed or was canceled.
func firstAttempt(r ghRun, jobs []ghJob) (outcome string, wall float64) {
	outcome = "skipped"
	var end time.Time
	for _, j := range jobs {
		if j.RunAttempt != 1 {
			continue
		}
		if j.CompletedAt.After(end) {
			end = j.CompletedAt
		}
		switch {
		case j.Conclusion == "failure" || j.Conclusion == "timed_out":
			outcome = "failure"
		case j.Conclusion == canceled && outcome != "failure":
			outcome = canceled
		case j.Conclusion == "success" && outcome == "skipped":
			outcome = "success"
		}
	}
	return outcome, end.Sub(r.CreatedAt).Minutes()
}

// rerunPassed is the lane and target of each job of a run that failed in
// one attempt and passed in a later one.
func rerunPassed(jobs []ghJob) []laneTarget {
	failedIn := map[string]int{}
	for _, j := range jobs {
		if j.Conclusion == "failure" && (failedIn[j.Name] == 0 || j.RunAttempt < failedIn[j.Name]) {
			failedIn[j.Name] = j.RunAttempt
		}
	}
	var passed []laneTarget
	for _, j := range jobs {
		if a, ok := failedIn[j.Name]; ok && j.Conclusion == "success" && j.RunAttempt > a {
			lane, target := jobLane(j.Name)
			passed = append(passed, laneTarget{lane, target})
			delete(failedIn, j.Name)
		}
	}
	return passed
}

// percentile is the p-th of xs by nearest rank.
func percentile(xs []float64, p float64) float64 {
	xs = slices.Sorted(slices.Values(xs))
	return xs[min(len(xs)-1, int(p*float64(len(xs))))]
}

func spread(xs []float64) string {
	if len(xs) == 0 {
		return "no runs"
	}
	return fmt.Sprintf("p50 %.1f min, p90 %.1f min, %d runs", percentile(xs, 0.5), percentile(xs, 0.9), len(xs))
}

func (s *stats) write(w io.Writer, days int) error {
	var b strings.Builder
	period := fmt.Sprintf("%d days", days)
	if days == 1 {
		period = "day"
	}
	fmt.Fprintf(&b, "## CI health\n\nFrom the runs of PR checks, Push checks and Daily checks in the last %s.\n\n", period)
	b.WriteString("| Measure | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| Wave check wall time, first attempts that passed | %s |\n", spread(s.waveGreen))
	fmt.Fprintf(&b, "| Wave check wall time, first attempts that failed | %s |\n", spread(s.waveRed))
	merge := "no runs"
	if s.mergeRuns > 0 {
		merge = fmt.Sprintf("%d of %d (%d%%)", s.mergeGreen, s.mergeRuns, 100*s.mergeGreen/s.mergeRuns)
	}
	fmt.Fprintf(&b, "| Merge checks that passed on their first attempt | %s |\n", merge)
	streak := "no runs"
	switch s.streakConclusion {
	case "success":
		streak = fmt.Sprintf("passed %d in a row", s.streak)
	case "failure":
		streak = fmt.Sprintf("failed %d in a row", s.streak)
	}
	fmt.Fprintf(&b, "| Daily check streak | %s |\n", streak)

	b.WriteString("\n### Failures per lane and target\n\n| Lane | Target | Jobs | Failed | Rate |\n|---|---|---|---|---|\n")
	for _, lt := range sortedLanes(s.lanes) {
		c := s.lanes[lt]
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %.1f%% |\n", lt.lane, cmp.Or(lt.target, "none"), c.jobs, c.failed, 100*float64(c.failed)/float64(c.jobs))
	}

	b.WriteString("\n### Flaky tests per lane and target\n\n")
	if s.artifacts > 0 {
		fmt.Fprintf(&b, "These counts come from the rerun labels in %d failures artifacts.\n\n", s.artifacts)
	} else {
		b.WriteString("No failures artifact could be read. These counts are jobs that failed and then passed when their run ran again.\n\n")
	}
	b.WriteString("| Lane | Target | Flaky |\n|---|---|---|\n")
	for _, lt := range sortedLanes(s.flaky) {
		fmt.Fprintf(&b, "| %s | %s | %d |\n", lt.lane, cmp.Or(lt.target, "none"), s.flaky[lt])
	}

	b.WriteString("\n### Queue time per runner OS\n\n| OS | Jobs | p50 | p90 |\n|---|---|---|---|\n")
	for _, o := range slices.Sorted(maps.Keys(s.queue)) {
		q := s.queue[o]
		fmt.Fprintf(&b, "| %s | %d | %.1f min | %.1f min |\n", o, len(q), percentile(q, 0.5), percentile(q, 0.9))
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("print the report: %w", err)
	}
	return nil
}

func sortedLanes[V any](m map[laneTarget]V) []laneTarget {
	return slices.SortedFunc(maps.Keys(m), func(a, b laneTarget) int {
		return cmp.Or(cmp.Compare(a.lane, b.lane), cmp.Compare(a.target, b.target))
	})
}
