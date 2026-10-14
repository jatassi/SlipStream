package main_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
)

func TestReadersWorkOnAStoppedRun(t *testing.T) {
	t.Parallel()
	runDir, _ := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	announce := "{fakes.tracker.url}/announce/nope?info_hash=aaaaaaaaaaaaaaaaaaaa&event=started&left=5&port=6881"
	if got, want := fakeworld(t, "exec", runDir, "--", self(t), commandArg, "get", "{ca.file}", announce), (result{0, "200\n", ""}); got != want {
		t.Fatalf("exec get = %+v, want %+v", got, want)
	}

	wait := fakeworld(t, "announces", "wait", runDir, "--where", "left=5")
	var rec events.Record
	if err := json.Unmarshal([]byte(wait.stdout), &rec); wait.code != 0 || err != nil {
		t.Fatalf("announces wait = %+v, want the announce", wait)
	}
	const data = `{"passkey":"nope","infoHash":"6161616161616161616161616161616161616161","peerId":"","port":6881,"event":"started","uploaded":0,"downloaded":0,"left":5,"failure":"passkey not recognized"}`
	if rec.Kind != "tracker.announce" || rec.Node != "world" || string(rec.Data) != data {
		t.Fatalf("announces wait = %s %s %s, want tracker.announce world %s", rec.Kind, rec.Node, rec.Data, data)
	}
	list := fakeworld(t, "announces", "list", runDir, "--node", "world", "--where", "left=5")
	if list != (result{0, wait.stdout, ""}) {
		t.Fatalf("announces list = %+v, want the announce wait printed", list)
	}

	resync := fakeworld(t, "sse", "list", runDir, "--as", "admin", "--event", "live.resync")
	var first events.Record
	if err := json.Unmarshal([]byte(resync.stdout), &first); resync.code != 0 || err != nil || first.Kind != "sse.message" || first.Node != "home" ||
		!strings.HasPrefix(string(first.Data), `{"as":"admin","node":"home","id":"`) || !strings.HasSuffix(string(first.Data), `","event":"live.resync","data":"{}"}`) {
		t.Fatalf("sse list = %+v, want the Admin's live.resync on home", resync)
	}

	fakeworld(t, "stop", runDir)
	if stopped := fakeworld(t, "announces", "list", runDir, "--node", "world", "--where", "left=5"); stopped != list {
		t.Fatalf("announces list on the stopped run = %+v, want %+v", stopped, list)
	}
	if stopped := fakeworld(t, "sse", "list", runDir, "--as", "admin", "--event", "live.resync"); stopped != resync {
		t.Fatalf("sse list on the stopped run = %+v, want %+v", stopped, resync)
	}
	missing := fakeworld(t, "announces", "wait", runDir, "--where", "left=6")
	if wantErr := "fakeworld announces wait: no record matches kind tracker.announce, left=6, after seq "; missing.code != 1 || !strings.HasPrefix(missing.stderr, wantErr) || !strings.Contains(missing.stderr, "\nthe last announces records:\n") {
		t.Fatalf("announces wait for no match on the stopped run = %+v, want exit 1 with %q and the last records", missing, wantErr)
	}
}
