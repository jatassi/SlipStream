package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"time"

	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/op"
	"github.com/jatassi/slipstream/internal/system"
)

// Mode is how the process runs the Node: run is headless, app is the
// desktop launch.
type Mode string

const (
	ModeRun Mode = "run"
	ModeApp Mode = "app"
)

// Phase is where the Node is in its start, which status answers before
// and after ready.
type Phase uint32

const (
	PhaseStarting Phase = iota
	// PhaseBackingUp is the Backup before a migration.
	PhaseBackingUp
	// PhaseRestoring is a restore finishing at start.
	PhaseRestoring
	PhaseReady
)

var phaseNames = [...]string{PhaseStarting: "starting", PhaseBackingUp: "backing up", PhaseRestoring: "restoring", PhaseReady: "ready"}

func (p Phase) String() string { return phaseNames[p] }

func (p Phase) MarshalText() ([]byte, error) { return []byte(p.String()), nil }

func (p *Phase) UnmarshalText(b []byte) error {
	for i, name := range phaseNames {
		if name == string(b) {
			*p = Phase(i)
			return nil
		}
	}
	return fmt.Errorf("phase %q isn't one of starting, backing up, restoring or ready", b)
}

func (n *Node) phase() Phase { return Phase(n.phaseOf.Load()) }

func (n *Node) setPhase(p Phase) { n.phaseOf.Store(uint32(p)) }

type StatusRequest struct{}

// StatusReply is the Node as a smoke test, service install --user, a
// second app launch or the tray sees it. Before ready it carries only who
// the process is and its Phase: Claim and URL are set once the Node is
// ready. Controller is set on a Worker. Tray and StartAtLogin are set in
// app mode only, StartAtLogin only where the OS has a login item. URL
// carries the Setup code while the Node is Unclaimed. StartedAt changes
// when the Node restarts, though PID may not.
type StatusReply struct {
	PID          int               `json:"pid"`
	StartedAt    time.Time         `json:"started_at"`
	Version      string            `json:"version"`
	Install      install.Kind      `json:"install"`
	Mode         Mode              `json:"mode"`
	Phase        Phase             `json:"phase"`
	Claim        *system.Claim     `json:"claim,omitempty"`
	Controller   *ControllerStatus `json:"controller,omitempty"`
	Tray         string            `json:"tray,omitempty"`
	StartAtLogin *bool             `json:"start_at_login,omitempty"`
	URL          string            `json:"url,omitempty"`
}

// ControllerStatus is a Worker's Controller: the name it gave at pairing
// and whether the link to it is up.
type ControllerStatus struct {
	Name      string `json:"name"`
	Reachable bool   `json:"reachable"`
}

var StatusOp = op.Op[StatusRequest, StatusReply]{Name: "status"}

func (n *Node) serveStatus(ctx context.Context, body json.RawMessage, send func(any) error) error {
	var req StatusRequest
	if err := control.Decode(body, &req); err != nil {
		return err
	}
	if p := n.phase(); p != PhaseReady {
		reply := n.process()
		reply.Phase = p
		return send(reply)
	}
	reply, err := n.Status(ctx)
	if err != nil {
		return err
	}
	return send(reply)
}

// process is the reply's fields that hold from the first moment of a
// start.
func (n *Node) process() StatusReply {
	return StatusReply{PID: os.Getpid(), StartedAt: n.started, Version: n.env.Version, Install: n.env.Install, Mode: n.env.Mode}
}

// Status is the ready Node as the status op and the tray report it.
func (n *Node) Status(ctx context.Context) (StatusReply, error) {
	claim, err := n.claim(ctx)
	if err != nil {
		return StatusReply{}, err
	}
	reply := n.process()
	reply.Phase = PhaseReady
	reply.Claim = &claim
	reply.URL = nodeURL(n.ready.Listeners.HTTP)
	switch claim {
	case system.Unclaimed:
		reply.URL = setupURL(n.ready.Listeners.HTTP, n.ready.SetupCode)
	case system.Worker:
		if rec, ok := n.pairing.Record(); ok {
			_, live := n.link.State(rec.Controller.Key).(link.Live)
			reply.Controller = &ControllerStatus{Name: rec.ControllerName, Reachable: live}
		}
	case system.Controller:
	}
	if n.env.Mode == ModeApp {
		if err := appStatus(&reply); err != nil {
			return StatusReply{}, err
		}
	}
	return reply, nil
}

func nodeURL(http netip.AddrPort) string {
	addr := http.Addr().Unmap()
	host := addr.String()
	switch {
	case addr.IsUnspecified() || addr.IsLoopback():
		host = "localhost"
	case addr.Is6():
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%d/", host, http.Port())
}

// setupURL is the setup page with the Setup code in the fragment, so the
// code never reaches a server log, a proxy or a Referer.
func setupURL(http netip.AddrPort, code string) string {
	return nodeURL(http) + "setup#code=" + code
}
