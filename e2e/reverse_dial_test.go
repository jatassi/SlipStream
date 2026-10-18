package e2e

import (
	"bufio"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// TestAControllerDialsANodeBehindCGNAT runs slipstream join for the
// running seedbox with a code naming home, whose link front refuses every
// dial. The command prints seedbox's Node address and waits; dialForCode
// on home pairs seedbox over home's session, and the command exits with
// the joined line. home dials seedbox from then on, across a restart of
// both Nodes.
func TestAControllerDialsANodeBehindCGNAT(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "cgnat-home")
	a := api(t, run)
	minted := a.MintJoinCode("home")
	cmd := exec.CommandContext(worldtest.Context(t), run.Manifest().Binary.Path, "join", "--data-dir", a.Node("seedbox").DataDir, minted.Code)
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot")}
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lines := bufio.NewScanner(stdout)
	var printed []string
	address, found := "", false
	for !found && lines.Scan() {
		printed = append(printed, lines.Text())
		address, found = strings.CutPrefix(lines.Text(), "  ")
	}
	if !strings.HasPrefix(address, "SLN1-") || !strings.HasPrefix(printed[0], "Can't reach the Controller at ") {
		t.Fatalf("slipstream join printed %q, want the Controller unreachable and a Node address", printed)
	}

	worldtest.Expect(t, "dialForCode", a.DialForCode("home", minted.ID, address), worldtest.Reply{Status: http.StatusAccepted, Body: ""})
	var last string
	for lines.Scan() {
		last = lines.Text()
	}
	if err := cmd.Wait(); err != nil || !strings.HasPrefix(last, "joined home as seedbox") {
		t.Fatalf("slipstream join exited %v after %q, want 0 after the joined line", err, last)
	}

	id := seedboxID(t, a)
	linked := linkUp(t, a, id, "")
	checkDialedByHome(t, run, a, id)
	act(t, run, "home", world.RestartNode)
	act(t, run, "seedbox", world.RestartNode)
	a = api(t, run)
	linkUp(t, a, id, linked)
	checkDialedByHome(t, run, a, id)
}

// seedboxID is the id home gave seedbox, which the manifest of a world
// that left seedbox Unclaimed doesn't hold.
func seedboxID(t *testing.T, a *worldtest.API) string {
	t.Helper()
	rows := a.Nodes("home")
	i := slices.IndexFunc(rows, func(r worldtest.Row) bool { return r.Name == "seedbox" })
	if i < 0 {
		t.Fatalf("listNodes on home has no row seedbox: %+v", rows)
	}
	return rows[i].ID
}

// checkDialedByHome checks home dialed the session it lists the Node id
// up on, at the address of seedbox's link front.
func checkDialedByHome(t *testing.T, run *world.Run, a *worldtest.API, id string) {
	t.Helper()
	front := run.Manifest().Links["seedbox"]
	if d := row(t, a.Nodes("home"), id).Dial; d.By != "controller" || !slices.Equal(d.ControllerDials, []string{front}) {
		t.Fatalf("home lists seedbox dialed by %q at %v, want dialed by the controller at seedbox's link front %s", d.By, d.ControllerDials, front)
	}
}
