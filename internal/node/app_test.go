package node

import (
	"testing"

	"github.com/jatassi/slipstream/internal/system"
)

func TestTrayLine(t *testing.T) {
	for _, c := range []struct {
		reply StatusReply
		want  string
	}{
		{StatusReply{Claim: system.Unclaimed}, "Unclaimed · open SlipStream to set up"},
		{StatusReply{Claim: system.Controller}, "Controller · running"},
		{StatusReply{Claim: system.Worker, Controller: &ControllerStatus{Name: "home", Reachable: true}}, "Worker of home · running"},
		{StatusReply{Claim: system.Worker, Controller: &ControllerStatus{Name: "home"}}, "Worker of home · Controller unreachable"},
		{StatusReply{Claim: system.Worker}, "Worker · joining"},
	} {
		if got := trayLine(c.reply); got != c.want {
			t.Errorf("trayLine(%+v) = %q, want %q", c.reply, got, c.want)
		}
	}
}
