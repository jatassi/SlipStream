package world_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
)

func TestLiveFaultsDropsARuleOnceItPlaysItsTimes(t *testing.T) {
	t.Parallel()
	runDir := t.TempDir()
	log, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	for _, r := range []struct{ kind, data string }{
		{events.FaultSet, `{"id":"once","target":"tmdb","fault":"status=503","times":1}`},
		{events.FaultSet, `{"id":"twice","target":"tmdb","fault":"status=503","times":2}`},
		{events.FaultSet, `{"id":"held","target":"seeders","fault":"throttle=16KiB/s","times":1}`},
		{events.FaultSet, `{"id":"reset","target":"seeders","fault":"reset","times":1}`},
		{events.FaultSet, `{"id":"always","target":"tmdb","fault":"status=500"}`},
		{events.FaultFired, `{"rule":"once","target":"tmdb","fault":"status=503","match":1}`},
		{events.FaultFired, `{"rule":"twice","target":"tmdb","fault":"status=503","match":1}`},
		{events.FaultFired, `{"rule":"held","target":"seeders","fault":"throttle=16KiB/s","match":1}`},
		{events.FaultFired, `{"rule":"reset","target":"seeders","fault":"reset","match":1}`},
		{events.FaultFired, `{"rule":"always","target":"tmdb","fault":"status=500","match":1}`},
		{events.FaultSet, `{"id":"once","target":"tmdb","fault":"status=429","times":1}`},
	} {
		if _, err := log.Append(events.Record{Kind: r.kind, Node: events.World, Data: json.RawMessage(r.data)}); err != nil {
			t.Fatal(err)
		}
	}
	live, err := world.LiveFaults(runDir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, rec := range live {
		var d struct{ ID string }
		if err := json.Unmarshal(rec.Data, &d); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, d.ID)
	}
	if want := []string{"twice", "held", "always", "once"}; !slices.Equal(ids, want) {
		t.Fatalf("live faults = %v, want %v", ids, want)
	}
}
