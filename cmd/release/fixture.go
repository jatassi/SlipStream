package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ReleasesDir holds one Backup per release, each written by that release's
// binary, which TestEveryReleasedBackupRestores in internal/node restores.
const releasesDir = "internal/backups/testdata/releases"

// Fixture is a release's Backup and what opens it. The Admin's email and
// PIN sign in once it is restored.
type Fixture struct {
	Version    string `json:"version"`
	Backup     string `json:"backup"`
	Passphrase string `json:"passphrase"`
	Email      string `json:"email"`
	PIN        string `json:"pin"`
}

const (
	fixtureEmail = "ada@example.test"
	fixturePIN   = "2468"
)

// writeReleaseBackup starts binary on a fresh data dir, claims it as a
// Controller, takes a Backup with slipstream backup, reads the passphrase
// in the Admin's session and writes the Backup and its Fixture into
// <root>/releasesDir/<version>/.
func writeReleaseBackup(ctx context.Context, root, binary string) (string, error) {
	dir, err := os.MkdirTemp("", "release-backup")
	if err != nil {
		return "", fmt.Errorf("make a data dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	data := filepath.Join(dir, "data")
	env := []string{
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"SLIPSTREAM_HTTP_LISTEN=127.0.0.1:0", "SLIPSTREAM_LINK_LISTEN=127.0.0.1:0", "SLIPSTREAM_TORRENT_LISTEN=127.0.0.1:0",
		"SLIPSTREAM_ENDPOINTS_INDEXERS_CATALOG=https://127.0.0.1:9/master",
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	run := exec.CommandContext(ctx, binary, "run", "--data-dir", data)
	run.Env = env
	out, err := run.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("start %s: %w", binary, err)
	}
	if err := run.Start(); err != nil {
		return "", fmt.Errorf("start %s: %w", binary, err)
	}
	defer func() {
		stop := exec.CommandContext(context.WithoutCancel(ctx), binary, "stop", "--data-dir", data)
		stop.Env = env
		_ = stop.Run()
		_ = run.Wait()
	}()
	ready, err := readReady(out)
	if err != nil {
		return "", err
	}
	go func() { _, _ = io.Copy(io.Discard, out) }()
	base := "http://" + ready.Listeners.HTTP
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("make a cookie jar: %w", err)
	}
	client := &http.Client{Jar: jar, Timeout: time.Minute}
	claim := fmt.Sprintf(`{"email":%q,"displayName":"Ada Okafor","pin":%q}`, fixtureEmail, fixturePIN)
	if err := post(ctx, client, base+"/api/v1/setup/controller", ready.SetupCode, claim); err != nil {
		return "", fmt.Errorf("claim the Controller: %w", err)
	}
	backup := exec.CommandContext(ctx, binary, "backup", "--data-dir", data)
	backup.Env = env
	printed, err := backup.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("slipstream backup: %w: %s", err, printed)
	}
	line := strings.TrimSpace(string(printed))
	path := line[strings.LastIndex(line, ": ")+2:]
	var shown struct {
		Passphrase string `json:"passphrase"`
	}
	if err := get(ctx, client, base+"/api/v1/backups/passphrase", &shown); err != nil {
		return "", fmt.Errorf("read the passphrase: %w", err)
	}
	f := Fixture{Version: ready.Version, Backup: filepath.Base(path), Passphrase: shown.Passphrase, Email: fixtureEmail, PIN: fixturePIN}
	into := filepath.Join(root, filepath.FromSlash(releasesDir), f.Version)
	if err := os.MkdirAll(into, 0o755); err != nil {
		return "", fmt.Errorf("make %s: %w", into, err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read the Backup: %w", err)
	}
	if err := os.WriteFile(filepath.Join(into, f.Backup), b, 0o644); err != nil {
		return "", fmt.Errorf("write the Backup: %w", err)
	}
	j, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode the fixture: %w", err)
	}
	if err := os.WriteFile(filepath.Join(into, "fixture.json"), append(j, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write the fixture: %w", err)
	}
	return into, nil
}

type readyRecord struct {
	Version   string `json:"version"`
	SetupCode string `json:"setup_code"`
	Listeners struct {
		HTTP string `json:"http"`
	} `json:"listeners"`
}

func readReady(r io.Reader) (readyRecord, error) {
	lines := bufio.NewScanner(r)
	lines.Buffer(make([]byte, 0, 1<<16), 1<<20)
	for lines.Scan() {
		var rec struct {
			Msg   string       `json:"msg"`
			Ready *readyRecord `json:"ready"`
		}
		if json.Unmarshal(lines.Bytes(), &rec) == nil && rec.Msg == "ready" && rec.Ready != nil {
			return *rec.Ready, nil
		}
	}
	return readyRecord{}, errors.New("the Node exited before it was ready")
}

func post(ctx context.Context, c *http.Client, url, setupCode, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("make the request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Setup-Code", setupCode)
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("send the request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s answered %d: %s", url, resp.StatusCode, bytes.TrimSpace(b))
	}
	return nil
}

func get(ctx context.Context, c *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("make the request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("send the request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("read %s: %w", url, err)
	}
	return nil
}
