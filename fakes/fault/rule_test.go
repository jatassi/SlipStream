package fault_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
)

func TestARuleRoundTripsThroughJSONWithItsFaultAsText(t *testing.T) {
	r := fault.Rule{
		ID: "lanterns", Target: "music-meta", Origin: "home", Route: "artist", Params: map[string]string{"mbid": "6d0f"},
		Fault: fault.Status{Code: 503, RetryAfter: time.Second}, Skip: 1, Times: 2,
		After: &events.Query{Kind: "tracker.announce", Match: map[string]string{"event": "completed"}},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"lanterns","target":"music-meta","origin":"home","route":"artist","params":{"mbid":"6d0f"},"skip":1,"times":2,` +
		`"after":{"kind":"tracker.announce","match":{"event":"completed"}},"fault":"status=503,retry-after=1s"}`
	if string(data) != want {
		t.Fatalf("Marshal = %s\nwant %s", data, want)
	}
	var back fault.Rule
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, r) {
		t.Fatalf("Unmarshal = %+v, want %+v", back, r)
	}
	if data, err := json.Marshal(fault.Rule{ID: "x", Target: "seeders", Fault: fault.Throttle(16 << 10)}); err != nil || string(data) != `{"id":"x","target":"seeders","fault":"throttle=16KiB/s"}` {
		t.Fatalf("Marshal of a bare rule = %s, %v", data, err)
	}
}

func TestARuleRefusesJSONItCannotParse(t *testing.T) {
	for _, c := range []struct{ json, want string }{
		{`{"id":"x","target":"tmdb","fault":"slow"}`, `rule "x": unknown fault "slow"`},
		{`{"id":"x","target":"tmdb"}`, `rule "x": unknown fault ""`},
		{`{"id":"x","target":"tmdb","fault":"hang","when":"now"}`, `rule: json: unknown field "when"`},
		{`{"id":"x","target":"tmdb","fault":"hang","after":{"kind":"a","seq":1}}`, `rule: json: unknown field "seq"`},
	} {
		var r fault.Rule
		if err := json.Unmarshal([]byte(c.json), &r); err == nil || err.Error() != c.want {
			t.Errorf("Unmarshal(%s) = %v, want %s", c.json, err, c.want)
		}
	}
}

func TestSetReturnsWhatArmingPlaysOnOpenConnections(t *testing.T) {
	var heard firings
	p := fault.NewPlan(fronts, heard.add)
	_ = p.Conn(pipeEnd(t), linkFront)
	fired, err := p.Set(fault.Rule{ID: "down", Target: "link/home", Fault: fault.Stall{}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []fault.Firing{{Rule: "down", Front: linkFront, Fault: fault.Stall{}, Match: 1}}; !reflect.DeepEqual(fired, want) {
		t.Fatalf("Set = %+v, want %+v", fired, want)
	}
	if got := heard.sorted(); len(got) != 0 {
		t.Fatalf("fired to the callback %q; arming's occurrences belong to Set", got)
	}
}
