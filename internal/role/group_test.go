package role_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
)

func TestEveryRoleIsInItsGlossaryGroup(t *testing.T) {
	var got []string
	for _, n := range role.Names() {
		got = append(got, string(n)+":"+string(n.Group()))
	}
	want := []string{
		"searching:acquire", "downloading:acquire", "storage:store", "converting:process", "subtitles:process",
		"metadata:connect", "media-server-link:connect", "notifying:connect", "relaying:reach", "serving:reach",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("groups = %v, want %v", got, want)
	}
	if got := role.Groups(); !slices.Equal(got, []role.Group{"acquire", "store", "process", "connect", "reach"}) {
		t.Fatalf("Groups() = %v", got)
	}
}
