// Package smoke installs each package a runner built the way a person
// would, checks the result from outside through files, logs, the status
// op, exit codes and OS state, then uninstalls it. Every recipe first
// removes what an earlier run left, so a rerun converges.
package smoke

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/cmd/release/internal/pack"
	"github.com/jatassi/slipstream/internal/install"
)

// Run is one smoke of the packages in Out.
type Run struct {
	Out      string
	Manifest pack.Manifest
	Log      io.Writer
}

var recipes = map[install.Kind]func(ctx context.Context, r *Run, files []string) error{
	install.MSI:      smokeMSI,
	install.ZIP:      smokeZIP,
	install.App:      smokeApp,
	install.Deb:      smokeDeb,
	install.RPM:      smokeRPM,
	install.AppImage: smokeAppImage,
	install.Tarball:  smokeTarball,
	install.Docker:   smokeDocker,
}

// Kinds lists every Kind with a smoke recipe.
func Kinds() []install.Kind { return slices.Sorted(maps.Keys(recipes)) }

// Smoke runs the recipe of every Kind the manifest lists and reports each
// that failed.
func Smoke(ctx context.Context, r *Run) error {
	var failed []error
	for _, k := range slices.Sorted(maps.Keys(r.Manifest.Packages)) {
		smoke, ok := recipes[k]
		if !ok {
			failed = append(failed, fmt.Errorf("%s: no smoke recipe", k))
			continue
		}
		r.logf("smoke %s\n", k)
		if err := smoke(ctx, r, r.Manifest.Packages[k]); err != nil {
			r.logf("smoke %s failed: %v\n", k, err)
			failed = append(failed, fmt.Errorf("%s: %w", k, err))
			continue
		}
		r.logf("smoke %s passed\n", k)
	}
	return errors.Join(failed...)
}

// logf writes a progress line to the log, which is only ever the
// terminal, so a failed write has nobody to tell.
func (r *Run) logf(format string, args ...any) { _, _ = fmt.Fprintf(r.Log, format, args...) }

func (r *Run) path(name string) (string, error) {
	p, err := filepath.Abs(filepath.Join(r.Out, name))
	if err != nil {
		return "", fmt.Errorf("find %s: %w", name, err)
	}
	return p, nil
}

// result is a finished command as a person at a terminal sees it.
type result struct {
	code           int
	stdout, stderr string
}

func (res result) String() string {
	return fmt.Sprintf("exit %d, stdout %q, stderr %q", res.code, strings.TrimSpace(res.stdout), strings.TrimSpace(res.stderr))
}

// exec runs argv to its end, with env added to the process's.
func (r *Run) exec(ctx context.Context, env []string, argv ...string) (result, error) {
	name, args := argv[0], argv[1:]
	cmd := exec.CommandContext(ctx, name, args...)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, r.Log)
	cmd.Stderr = io.MultiWriter(&stderr, r.Log)
	r.logf("+ %s %s\n", name, strings.Join(args, " "))
	err := cmd.Run()
	res := result{stdout: stdout.String(), stderr: stderr.String()}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		res.code = exit.ExitCode()
	case err != nil:
		return res, fmt.Errorf("%s: %w", name, err)
	}
	return res, nil
}

// must runs argv, which has to exit 0, and returns its stdout.
func (r *Run) must(ctx context.Context, env []string, argv ...string) (string, error) {
	res, err := r.exec(ctx, env, argv...)
	if err != nil {
		return "", err
	}
	if res.code != 0 {
		return "", fmt.Errorf("%s: %s", strings.Join(argv, " "), res)
	}
	return res.stdout, nil
}

// expect runs argv, which has to exit with code and print want on stderr.
func (r *Run) expect(ctx context.Context, code int, want string, argv ...string) error {
	res, err := r.exec(ctx, nil, argv...)
	if err != nil {
		return err
	}
	if res.code != code || !strings.Contains(res.stderr, want) {
		return fmt.Errorf("%s: %s; want exit %d and %q on stderr", strings.Join(argv, " "), res, code, want)
	}
	return nil
}

// tidy runs a cleanup command whose failure only means there was nothing
// to clean.
func (r *Run) tidy(ctx context.Context, argv ...string) {
	_, _ = r.exec(ctx, nil, argv...)
}

// status is the status op's reply, as slipstream status prints it.
type status struct {
	PID          int    `json:"pid"`
	Install      string `json:"install"`
	Mode         string `json:"mode"`
	Tray         string `json:"tray"`
	StartAtLogin *bool  `json:"start_at_login"`
	URL          string `json:"url"`
}

func parseStatus(out string) (status, error) {
	var s status
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		return status{}, fmt.Errorf("status printed %q: %w", out, err)
	}
	return s, nil
}

// statusOf runs argv, a command line ending in status.
func (r *Run) statusOf(ctx context.Context, argv ...string) (status, error) {
	out, err := r.must(ctx, nil, argv...)
	if err != nil {
		return status{}, err
	}
	return parseStatus(out)
}

// want checks a status against the install kind and mode a smoke expects.
func (s status) want(k install.Kind, mode string) error {
	if s.Install != k.String() || s.Mode != mode {
		return fmt.Errorf("status reports install %q and mode %q, want %q and %q", s.Install, s.Mode, k, mode)
	}
	return nil
}

// wantApp checks a Node a desktop launch started: app mode under kind k,
// its tray item up, and start at login as the package leaves it.
func (s status) wantApp(k install.Kind, startAtLogin bool) error {
	if err := s.want(k, "app"); err != nil {
		return err
	}
	if s.Tray != "ready" {
		return fmt.Errorf("status reports the tray %q, want ready", s.Tray)
	}
	if s.StartAtLogin == nil {
		return errors.New("status does not report start at login")
	}
	if *s.StartAtLogin != startAtLogin {
		return fmt.Errorf("status reports start at login %t, want %t", *s.StartAtLogin, startAtLogin)
	}
	return nil
}

// record is one JSON log line of the Node.
type record struct {
	Msg   string `json:"msg"`
	URL   string `json:"url"`
	Ready *struct {
		PID       int    `json:"pid"`
		SetupCode string `json:"setup_code"`
	} `json:"ready"`
}

// logged holds the records that matter to a smoke: the ready line and the
// setup page.
type logged struct {
	pid       int
	setupCode string
	setupURL  string
}

const waitFor = 90 * time.Second

// awaitReady polls read, which returns the Node's log so far, until it
// holds the ready record, and returns what the log says.
func awaitReady(ctx context.Context, what string, read func() (string, error)) (logged, error) {
	deadline := time.Now().Add(waitFor)
	for {
		text, err := read()
		if err == nil {
			if l, ok := parseLog(text); ok {
				return l, nil
			}
		}
		if time.Now().After(deadline) {
			return logged{}, fmt.Errorf("no ready record in %s within %s (last read: %w)", what, waitFor, err)
		}
		select {
		case <-ctx.Done():
			return logged{}, fmt.Errorf("wait for %s: %w", what, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func parseLog(text string) (logged, bool) {
	var l logged
	ready := false
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		var rec record
		if json.Unmarshal(sc.Bytes(), &rec) != nil {
			continue
		}
		switch {
		case rec.Msg == "setup page":
			l.setupURL = rec.URL
		case rec.Msg == "ready" && rec.Ready != nil:
			l.pid, l.setupCode, ready = rec.Ready.PID, rec.Ready.SetupCode, true
		}
	}
	return l, ready
}

func readFile(path string) func() (string, error) {
	return func() (string, error) {
		b, err := os.ReadFile(path)
		return string(b), err
	}
}

// awaitFile polls until path holds text containing want.
func awaitFile(ctx context.Context, path, want string) error {
	deadline := time.Now().Add(waitFor)
	for {
		b, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(b), want) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s does not hold %q within %s (read %q, %w)", path, want, waitFor, b, err)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for %s: %w", path, ctx.Err())
		case <-time.After(time.Second):
		}
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func mustExist(paths ...string) error {
	for _, p := range paths {
		if !exists(p) {
			return fmt.Errorf("%s does not exist", p)
		}
	}
	return nil
}

// layout is this process's facts for install's path rules.
func layout(goos string) install.Layout {
	return install.Layout{GOOS: goos, Environ: os.Environ()}
}
