package node

import (
	"testing"

	"github.com/jatassi/slipstream/internal/system"
)

func claimed(c system.Claim) *system.Claim { return &c }

func TestTrayLine(t *testing.T) {
	for _, c := range []struct {
		reply StatusReply
		want  string
	}{
		{StatusReply{Claim: claimed(system.Unclaimed)}, "Not set up · open SlipStream to set it up"},
		{StatusReply{Claim: claimed(system.Controller)}, "Controller · running"},
		{StatusReply{Claim: claimed(system.Worker), Controller: &ControllerStatus{Name: "home", Reachable: true}}, "Worker of home · running"},
		{StatusReply{Claim: claimed(system.Worker), Controller: &ControllerStatus{Name: "home"}}, "Worker of home · Controller unreachable"},
		{StatusReply{Claim: claimed(system.Worker)}, "Worker · joining"},
	} {
		if got := trayLine(c.reply); got != c.want {
			t.Errorf("trayLine(%+v) = %q, want %q", c.reply, got, c.want)
		}
	}
}
