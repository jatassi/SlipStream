package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestTheTierJournalsItsCallsAndWaits(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	m := run.Manifest()
	a := api(t, run)
	worldtest.Expect(t, "GET /api/v1/status", a.Get("home", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"controller","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})
	polls := 0
	worldtest.Until(t, "home answers twice", func() (bool, error) {
		polls++
		return a.Admin("home", http.MethodGet, "/api/v1/system", "").Status == 200 && polls == 2, nil
	})

	recs, err := events.Select(m.RunDir, events.Query{Kind: "step.*"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range recs {
		got = append(got, fmt.Sprintf("%s %s %v", r.Kind, r.Data, r.Files))
	}
	want := []string{
		`step.start {"n":1,"verb":"exec","args":["--","curl","-sS","{nodes.home.url}/api/v1/status"]} map[]`,
		`step.end {"n":1,"exit":0} map[stdout:journal/1.out]`,
		`step.start {"n":2,"note":"until home answers twice"} map[]`,
		`step.end {"n":2,"exit":0} map[]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the journal holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if out, err := os.ReadFile(filepath.Join(m.RunDir, "journal", "1.out")); err != nil || string(out) != `{"claim":"controller","passkeys":{"available":false,"reason":"ip_address"}}`+"\n" {
		t.Fatalf("journal/1.out = %q, %v", out, err)
	}
	argv, err := world.Expand(m, []string{"{nodes.home.url}/api/v1/status"})
	if err != nil || argv[0] != m.Nodes[0].URL+"/api/v1/status" {
		t.Fatalf("the journaled URL expands to %q, %v", argv, err)
	}
}
