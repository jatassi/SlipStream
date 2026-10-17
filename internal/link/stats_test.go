package link_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
)

// Each side counts the bytes its TCP connection carried, so one side's
// sent is the other's received, and each measures at least the 80 ms
// round trip the path adds.
func TestStatsCountTheSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lockstep: lockstep(calls)}, linktest.Side{})
		if _, err := p.Net.Path(linktest.WorkerAddr, linktest.ControllerAddr).Set(fault.Rule{Fault: fault.Latency(40 * time.Millisecond)}); err != nil {
			t.Fatal(err)
		}
		if _, ok := p.Worker.Stats(p.Controller.Key); ok {
			t.Fatal("the Worker has stats before any session")
		}
		p.Connect(t)
		if _, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); err != nil {
			t.Fatal(err)
		}
		<-calls
		time.Sleep(time.Second)
		synctest.Wait()

		c, cok := p.Controller.Stats(p.Worker.Key)
		w, wok := p.Worker.Stats(p.Controller.Key)
		if !cok || !wok || c.Sent == 0 || c.Sent != w.Received || c.Received != w.Sent {
			t.Fatalf("the Controller counts %+v and the Worker %+v, want each side's sent to be the other's received", c, w)
		}
		if c.RTT < 80*time.Millisecond || w.RTT < 80*time.Millisecond || c.RTT > time.Second || w.RTT > time.Second {
			t.Fatalf("the Controller measures a %v round trip and the Worker %v, want at least the path's 80ms", c.RTT, w.RTT)
		}

		before := w.Sent
		if _, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); err != nil {
			t.Fatal(err)
		}
		<-calls
		if w, _ := p.Worker.Stats(p.Controller.Key); w.Sent <= before {
			t.Fatalf("after another call the Worker has sent %d bytes, as before it", w.Sent)
		}
	})
}
