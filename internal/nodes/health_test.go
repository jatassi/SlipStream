package nodes

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/role"
)

func fits(at time.Time, ps ...role.Probe) []role.ProbeResult {
	out := make([]role.ProbeResult, len(ps))
	for i, p := range ps {
		out[i] = role.ProbeResult{Probe: p, Fit: role.Fits, At: at}
	}
	return out
}

// everyFit is a verdict that fits for every probe, on both Nodes.
func everyFit() map[ID][]role.ProbeResult {
	return map[ID][]role.ProbeResult{
		home: fits(start, role.Probes()...),
		box:  fits(start, role.Probes()...),
	}
}

func with(probes map[ID][]role.ProbeResult, node ID, p role.ProbeResult) map[ID][]role.ProbeResult {
	out := maps.Clone(probes)
	out[node] = slices.Clone(out[node])
	for i := range out[node] {
		if out[node][i].Probe == p.Probe {
			out[node][i] = p
		}
	}
	return out
}

func key(parts ...string) health.Key { return health.KeyOf(parts...) }

// awayAt is the cluster with seedbox last seen and gone an hour in, at
// the given time after it left.
func awayAt(after time.Duration) *Board {
	in := cluster()
	in.node(box).seen = start.Add(time.Hour)
	in.detach(box, start.Add(time.Hour))
	return in.at(start.Add(time.Hour + after))
}

func TestTheNodesItemsFollowTheBoardAndTheProbes(t *testing.T) {
	failing := role.ProbeResult{Probe: role.EngineStarts, Fit: role.Fails, Reason: role.DownloadsNotWritable, At: start}
	weak := role.ProbeResult{Probe: role.EngineStarts, Fit: role.Weak, Reason: role.DownloadsOnNetworkShare, At: start}
	mismatched := cluster()
	mismatched.node(box).attached, mismatched.node(box).mismatched = false, true
	for _, c := range []struct {
		name   string
		board  *Board
		probes map[ID][]role.ProbeResult
		items  map[health.Key]health.Level
		unsure []health.Key
	}{
		{
			name:  "every Node in step with fitting probes",
			board: cluster().at(start), probes: everyFit(),
		},
		{
			name:  "no verdicts since the Controller started",
			board: cluster().at(start), probes: map[ID][]role.ProbeResult{home: fits(start, role.DataWritable)},
			unsure: []health.Key{key(home.String(), "listener-up"), key(box.String(), "engine-starts")},
		},
		{
			name:  "a failing probe for a Role the Node holds",
			board: cluster().at(start), probes: with(everyFit(), box, failing),
			items: map[health.Key]health.Level{key(box.String(), "engine-starts"): health.Fail},
		},
		{
			name:  "an engine that can't listen, which Download Clients reports",
			board: cluster().at(start), probes: with(everyFit(), box, role.ProbeResult{Probe: role.EngineStarts, Fit: role.Fails, Reason: role.EngineNotListening, At: start}),
		},
		{
			name:  "a weak probe for a Role the Node holds",
			board: cluster().at(start), probes: with(everyFit(), box, weak),
			items: map[health.Key]health.Level{key(box.String(), "engine-starts"): health.Warn},
		},
		{
			name:  "a failing probe for a Role the Node doesn't hold",
			board: cluster().at(start), probes: with(everyFit(), home, failing),
		},
		{
			name:  "a Node reconnecting",
			board: awayAt(30 * time.Second), probes: with(everyFit(), box, failing),
			unsure: []health.Key{key(box.String()), key(box.String(), "engine-starts")},
		},
		{
			name:  "a Node offline inside its grace",
			board: awayAt(2 * time.Minute), probes: everyFit(),
			unsure: []health.Key{key(box.String()), key(box.String(), "engine-starts")},
		},
		{
			name:  "a Node offline past its grace",
			board: awayAt(15 * time.Minute), probes: everyFit(),
			items:  map[health.Key]health.Level{key(box.String()): health.Fail, key("covering", "searching"): health.Warn},
			unsure: []health.Key{key(box.String(), "engine-starts")},
		},
		{
			name:  "a Node on another version",
			board: mismatched.at(start), probes: everyFit(),
			items:  map[health.Key]health.Level{key(box.String()): health.Fail},
			unsure: []health.Key{key(box.String(), "engine-starts")},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := itemsOf(c.board, c.probes)
			if items, unsure := keysOf(a); !maps.Equal(items, c.items) || !slices.Equal(unsure, c.unsure) {
				t.Errorf("items %v and unsure %v, want %v and %v", items, unsure, c.items, c.unsure)
			}
			for _, s := range says(a) {
				if strings.Contains(s.Subject+s.Text, box.String()) || strings.Contains(s.Subject+s.Text, home.String()) {
					t.Errorf("%q: %q names a Node by its id", s.Subject, s.Text)
				}
			}
		})
	}
}

func keysOf(a health.Answer) (items map[health.Key]health.Level, unsure []health.Key) {
	items = map[health.Key]health.Level{}
	for _, it := range a.Items {
		items[it.Key] = it.Level
	}
	for _, u := range a.Unsure {
		unsure = append(unsure, u.Key)
	}
	return items, unsure
}

func says(a health.Answer) []health.Says {
	var out []health.Says
	for _, it := range a.Items {
		out = append(out, it.Says)
	}
	for _, u := range a.Unsure {
		out = append(out, u.Says)
	}
	return out
}

func TestTheNodesItemsSayWhatToDo(t *testing.T) {
	lapsed := itemsOf(awayAt(15*time.Minute), everyFit())
	want := []health.Item{
		{Key: key(box.String()), Level: health.Fail, Says: health.Says{
			Subject: "seedbox",
			Text:    "seedbox has been offline past the offline grace. Start SlipStream on it, or remove it if it is gone.",
			Detail:  []health.Fact{{Label: "Offline since", At: start.Add(time.Hour)}},
			Action:  health.LinkAction("/nodes/"+box.String(), "Open seedbox"),
		}},
		{Key: key("covering", "searching"), Level: health.Warn, Says: health.Says{
			Subject: "Searching on home",
			Text:    "home searches in place of seedbox. Bring seedbox back, or give Searching to another Node.",
			Action:  health.LinkAction("/nodes/"+box.String(), "Open seedbox"),
		}},
	}
	if !slices.EqualFunc(lapsed.Items, want, sameItem) {
		t.Errorf("past the grace the items are %+v, want %+v", lapsed.Items, want)
	}
	failing := role.ProbeResult{Probe: role.EngineStarts, Fit: role.Fails, Reason: role.DownloadsNotWritable, At: start}
	probed := itemsOf(cluster().at(start), with(everyFit(), box, failing))
	if got, want := probed.Items, []health.Item{{Key: key(box.String(), "engine-starts"), Level: health.Fail, Says: health.Says{
		Subject: "Downloading on seedbox",
		Text:    "Downloading can't run on seedbox. Its downloads folder can't be written to. Let SlipStream write to its downloads folder, then Check now.",
		Detail:  []health.Fact{{Label: "Probed", At: start}},
		Action:  health.CheckAction(),
	}}}; !slices.EqualFunc(got, want, sameItem) {
		t.Errorf("a failing probe's items are %+v, want %+v", got, want)
	}
}

func sameItem(a, b health.Item) bool {
	return a.Key == b.Key && a.Level == b.Level && a.Subject == b.Subject && a.Text == b.Text && a.Action == b.Action &&
		slices.EqualFunc(a.Detail, b.Detail, func(x, y health.Fact) bool { return x.Label == y.Label && x.Text == y.Text && x.At.Equal(y.At) })
}

func TestEveryProbeReasonSaysWhatToDo(t *testing.T) {
	for _, r := range role.ProbeReasons() {
		if probeFixes[r] == "" {
			t.Errorf("probe reason %s has no fix", r)
		}
	}
}
