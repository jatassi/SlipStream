package nodes

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/role"
)

func TestTheSentenceSaysWhatStandsOut(t *testing.T) {
	crateIn := nodeIn{id: crate, name: "crate", assigned: role.Of(role.Searching), attached: true, since: start, seen: start}
	for _, tc := range []struct {
		name string
		edit func(in *inputs)
		at   time.Duration
		want string
	}{
		{"one Node", func(in *inputs) { in.nodes = in.nodes[:1] }, 0, "home is the only Node."},
		{"two in step", func(*inputs) {}, 0, "Both Nodes are online."},
		{"three in step", func(in *inputs) { in.nodes = append(in.nodes, crateIn) }, 0, "All 3 Nodes are online."},
		{"a blip", func(in *inputs) { in.detach(box, start) }, 30 * time.Second, "seedbox is reconnecting."},
		{"offline within the grace", func(in *inputs) { in.detach(box, start) }, 5 * time.Minute, "seedbox is offline."},
		{"lapsed and covered", func(in *inputs) { in.detach(box, start) }, 15 * time.Minute, "seedbox is offline. home is searching in its place."},
		{"two lapsed and covered", func(in *inputs) {
			in.nodes = append(in.nodes, crateIn)
			in.detach(box, start)
			in.detach(crate, start)
		}, 15 * time.Minute, "seedbox is offline. crate is offline. home is searching in their place."},
		{"back and settling", func(in *inputs) {
			in.detach(box, start)
			in.attach(box, start.Add(20*time.Minute))
		}, 20 * time.Minute, "seedbox is back and settling. home is searching in its place."},
		{"mismatched", func(in *inputs) {
			in.detach(box, start)
			in.node(box).mismatched = true
		}, 5 * time.Minute, "seedbox runs another version."},
		{"draining", func(in *inputs) {
			b := in.node(box)
			b.assigned, b.work = role.Of(role.Searching), role.Of(role.Downloading)
		}, 0, "seedbox is draining Downloading."},
		{"writing off", func(in *inputs) {
			b := in.node(box)
			b.assigned, b.work, b.forfeit = role.Of(role.Searching), role.Of(role.Downloading), role.Of(role.Downloading)
		}, 0, "seedbox is writing off Downloading."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := cluster()
			tc.edit(in)
			if got := sentence(in.at(start.Add(tc.at))); got != tc.want {
				t.Errorf("sentence = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEveryRoleHasWordsForWhatItsHolderIsDoing(t *testing.T) {
	want := map[role.Name]string{
		role.Searching:       "searching",
		role.Downloading:     "downloading",
		role.Storage:         "storing the library",
		role.Converting:      "converting",
		role.Subtitles:       "fetching subtitles",
		role.Metadata:        "reading metadata",
		role.MediaServerLink: "talking to Media Servers",
		role.Notifying:       "sending mail",
		role.Relaying:        "relaying",
		role.Serving:         "serving SlipStream",
	}
	for _, r := range role.Names() {
		if got := doing(r); got != want[r] {
			t.Errorf("doing(%s) = %q, want %q", r, got, want[r])
		}
	}
	if got := doingAll(role.Of(role.Searching, role.Metadata, role.Notifying)); got != "searching, reading metadata and sending mail" {
		t.Errorf("doing three Roles reads %q", got)
	}
}
