package role_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
)

func TestEveryRoleHasItsGlossaryLabelAndGroup(t *testing.T) {
	var got []string
	for _, n := range role.Names() {
		got = append(got, n.Label()+":"+string(n.Group()))
	}
	want := []string{
		"Searching:acquire", "Downloading:acquire", "Storage:store", "Converting:process", "Subtitles:process",
		"Metadata:connect", "Media Server link:connect", "Notifying:connect", "Relaying:reach", "Serving:reach",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	if got := role.Groups(); !slices.Equal(got, []role.Group{"acquire", "store", "process", "connect", "reach"}) {
		t.Fatalf("Groups() = %v", got)
	}
	var labels []string
	for _, g := range role.Groups() {
		labels = append(labels, g.Label())
	}
	if want := []string{"Acquire", "Store", "Process", "Connect", "Reach"}; !slices.Equal(labels, want) {
		t.Fatalf("the Group labels are %v, want %v", labels, want)
	}
}
