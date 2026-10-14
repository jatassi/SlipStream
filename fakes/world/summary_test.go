package world_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
)

func writeRunFile(t *testing.T, runDir, rel, content string) {
	t.Helper()
	path := filepath.Join(runDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSummary(t *testing.T) {
	t.Parallel()
	runDir := t.TempDir()
	m := world.Manifest{
		RunID: "k3v9x2qa", RunDir: "/runs/k3v9x2qa",
		Scenario: world.ScenarioRef{Name: "home", SHA256: "2d69", Seed: "home"},
		Binary:   world.Binary{Path: "/runs/k3v9x2qa/bin/slipstream", SHA256: "3ebb", Version: "v0.1.0"},
	}
	data, err := json.Marshal(m) //nolint:gosec // a stand-in run's manifest
	if err != nil {
		t.Fatal(err)
	}
	writeRunFile(t, runDir, "manifest.stopped.json", string(data))
	writeRunFile(t, runDir, "journal/3.out", "ok   1 the run is ours\n")
	writeRunFile(t, runDir, "proof/search-and-grab/held/1.out", "503\n")
	log, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []events.Record{
		{Kind: events.RunLaunched, Node: "world", Data: json.RawMessage(`{"origin":{"kind":"test","command":"go test ./e2e -run '^TestSkeletonHome$' -count=1"}}`)},
		{Kind: events.StepStart, Node: "world", Data: json.RawMessage(`{"n":1,"verb":"exec","args":["--proof","search-and-grab/held","--","curl","-H","X-Api-Key: {apiKey}","{nodes.home.url}/a|b"]}`)},
		{Kind: events.NodeError, Node: "home", Data: json.RawMessage(`{"level":"ERROR","msg":"boom"}`)},
		{Kind: events.StepEnd, Node: "world", Data: json.RawMessage(`{"n":1,"exit":0}`), Files: map[string]string{"stdout": "proof/search-and-grab/held/1.out", "stderr": "proof/search-and-grab/held/1.err"}},
		{Kind: events.StepStart, Node: "world", Data: json.RawMessage(`{"n":2,"verb":"fault set","args":["hold","seeders@seedbox","throttle=16KiB/s"]}`)},
		{Kind: events.FaultSet, Node: "world", Data: json.RawMessage(`{"fault":"throttle=16KiB/s","id":"hold","origin":"seedbox","step":2,"target":"seeders"}`)},
		{Kind: events.StepEnd, Node: "world", Data: json.RawMessage(`{"n":2,"exit":0}`)},
		{Kind: events.StepStart, Node: "world", Data: json.RawMessage(`{"n":3,"verb":"doctor"}`)},
		{Kind: events.StepEnd, Node: "world", Data: json.RawMessage(`{"n":3,"exit":0}`), Files: map[string]string{"stdout": "journal/3.out"}},
		{Kind: events.NodeError, Node: "seedbox", Data: json.RawMessage(`{"level":"ERROR","msg":"late"}`)},
		{Kind: events.StepStart, Node: "world", Data: json.RawMessage(`{"n":4,"note":"until the Grab is imported"}`)},
	} {
		if _, err := log.Append(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := world.Summary(runDir)
	want := "# Run k3v9x2qa\n\n" +
		"- Build: v0.1.0, sha256 3ebb; bin/slipstream is not in this bundle\n" +
		"- Scenario: home (seed home, sha256 2d69)\n" +
		"- Origin: test, `go test ./e2e -run '^TestSkeletonHome$' -count=1`\n" +
		"\n## Doctor\n\nStep 3, exit 0:\n\n```text\nok   1 the run is ours\n```\n" +
		"\n## Steps\n\n| n | command | exit | proof |\n|---|---|---|---|\n" +
		"| 1 | `fakeworld exec --proof search-and-grab/held -- curl -H 'X-Api-Key: {apiKey}' '{nodes.home.url}/a\\|b'` | 0 | proof/search-and-grab/held |\n" +
		"| 2 | `fakeworld fault set hold seeders@seedbox throttle=16KiB/s` | 0 |  |\n" +
		"| 3 | `fakeworld doctor` | 0 |  |\n" +
		"| 4 | `note: until the Grab is imported` | none (it never ended) |  |\n" +
		"\n## Cues and faults\n\n| seq | kind | node | data |\n|---|---|---|---|\n" +
		"| 6 | fault.set | world | `{\"fault\":\"throttle=16KiB/s\",\"id\":\"hold\",\"origin\":\"seedbox\",\"step\":2,\"target\":\"seeders\"}` |\n" +
		"\n## Node errors\n\n" +
		"- seq 3, step 1, node home: `{\"level\":\"ERROR\",\"msg\":\"boom\"}`\n" +
		"- seq 10, no step, node seedbox: `{\"level\":\"ERROR\",\"msg\":\"late\"}`\n" +
		"\n## Proof files\n\n- proof/search-and-grab/held/1.out\n"
	if err != nil || got != want {
		t.Fatalf("Summary = %v\n%s\nwant\n%s", err, got, want)
	}
}
