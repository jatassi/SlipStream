package nodes

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/role"
)

// TestTheSentenceSaysWhatStandsOut checks the Nodes page's sentence names
// the Nodes that stand out, and only those, with the state each is in.
func TestTheSentenceSaysWhatStandsOut(t *testing.T) {
	crateIn := nodeIn{id: crate, name: "crate", assigned: role.Of(role.Searching), attached: true, since: start, seen: start}
	for _, tc := range []struct {
		name  string
		edit  func(in *inputs)
		at    time.Duration
		names []string
		state string
	}{
		{"one Node", func(in *inputs) { in.nodes = in.nodes[:1] }, 0, []string{"home"}, "only"},
		{"two in step", func(*inputs) {}, 0, nil, "online"},
		{"three in step", func(in *inputs) { in.nodes = append(in.nodes, crateIn) }, 0, nil, "3"},
		{"a blip", func(in *inputs) { in.detach(box, start) }, 30 * time.Second, []string{"seedbox"}, "reconnecting"},
		{"offline within the grace", func(in *inputs) { in.detach(box, start) }, 5 * time.Minute, []string{"seedbox"}, "offline"},
		{"lapsed and covered", func(in *inputs) { in.detach(box, start) }, 15 * time.Minute, []string{"home", "seedbox"}, "searching"},
		{"two lapsed and covered", func(in *inputs) {
			in.nodes = append(in.nodes, crateIn)
			in.detach(box, start)
			in.detach(crate, start)
		}, 15 * time.Minute, []string{"crate", "home", "seedbox"}, "searching"},
		{"back and settling", func(in *inputs) {
			in.detach(box, start)
			in.attach(box, start.Add(20*time.Minute))
		}, 20 * time.Minute, []string{"home", "seedbox"}, "back online"},
		{"mismatched", func(in *inputs) {
			in.detach(box, start)
			in.node(box).mismatched = true
		}, 5 * time.Minute, []string{"seedbox"}, "version"},
		{"draining", func(in *inputs) {
			b := in.node(box)
			b.assigned, b.work = role.Of(role.Searching), role.Of(role.Downloading)
		}, 0, []string{"seedbox"}, "finishing"},
		{"writing off", func(in *inputs) {
			b := in.node(box)
			b.assigned, b.work, b.forfeit = role.Of(role.Searching), role.Of(role.Downloading), role.Of(role.Downloading)
		}, 0, []string{"seedbox"}, "giving up"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := cluster()
			tc.edit(in)
			got := sentence(in.at(start.Add(tc.at)))
			var named []string
			for _, n := range []string{"crate", "home", "seedbox"} {
				if strings.Contains(got, n) {
					named = append(named, n)
				}
			}
			if !slices.Equal(named, tc.names) || !strings.Contains(got, tc.state) {
				t.Errorf("sentence = %q, want it to name %v and say %q", got, tc.names, tc.state)
			}
		})
	}
}
