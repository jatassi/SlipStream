package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ghRun is a workflow run as GitHub's Actions API lists it.
type ghRun struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	Event      string    `json:"event"`
	HeadBranch string    `json:"head_branch"`
	HeadSHA    string    `json:"head_sha"`
	Conclusion string    `json:"conclusion"`
	RunAttempt int       `json:"run_attempt"`
	CreatedAt  time.Time `json:"created_at"`
	HeadCommit struct {
		TreeID string `json:"tree_id"`
	} `json:"head_commit"`
	PullRequests []struct {
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_requests"`
}

// base is the branch the run's pull request merges into, empty for a run
// of no pull request in this repository.
func (r ghRun) base() string {
	if len(r.PullRequests) == 0 {
		return ""
	}
	return r.PullRequests[0].Base.Ref
}

// ghJob is a job of a run, as the Actions API lists it. A job that never
// started has a zero StartedAt.
type ghJob struct {
	ID          int64     `json:"id"`
	RunID       int64     `json:"run_id"`
	RunAttempt  int       `json:"run_attempt"`
	Name        string    `json:"name"`
	Conclusion  string    `json:"conclusion"`
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Labels      []string  `json:"labels"`
}

var targetName = regexp.MustCompile(`(linux|windows|darwin)-(amd64|arm64)`)

// jobLane is a job's lane and target as the workflows name its job, such
// as "Go tests (linux-amd64, shard 2 of 4)" under the names of the jobs
// that called its workflow. A job of no one target has an empty target.
func jobLane(name string) (lane, target string) {
	lane, rest, _ := strings.Cut(jobTitle(name), " (")
	return lane, targetName.FindString(rest)
}

// jobTitle is a job's name without the names of the jobs that called its
// workflow.
func jobTitle(name string) string {
	if i := strings.LastIndex(name, " / "); i >= 0 {
		return name[i+len(" / "):]
	}
	return name
}

// runnerOS names the OS of the runner a job asked for by its labels.
func runnerOS(labels []string) string {
	l := strings.Join(labels, ",")
	switch {
	case strings.Contains(l, "macos"):
		return "macOS"
	case strings.Contains(l, "windows"):
		return "Windows"
	case strings.Contains(l, "arm"):
		return "Linux arm64"
	default:
		return "Linux"
	}
}

// ghList reads the items of an Actions API list, every page of it when
// paginate is set. items is the field that holds them, such as
// .workflow_runs.
func ghList[T any](ctx context.Context, path, items string, paginate bool) ([]T, error) {
	args := []string{"api", "-X", "GET", path, "--jq", items + "[]"}
	if paginate {
		args = append(args, "--paginate")
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", path, err)
	}
	var list []T
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var v T
		if err := dec.Decode(&v); errors.Is(err, io.EOF) {
			return list, nil
		} else if err != nil {
			return nil, fmt.Errorf("read gh api %s: %w", path, err)
		}
		list = append(list, v)
	}
}

func runJobs(ctx context.Context, repo string, run int64, filter string) ([]ghJob, error) {
	return ghList[ghJob](ctx, fmt.Sprintf("repos/%s/actions/runs/%d/jobs?filter=%s&per_page=50", repo, run, filter), ".jobs", true)
}
