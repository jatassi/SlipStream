// Package systemapi holds the Node's liveness, claim state and runtime
// facts.
package systemapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/system"
)

type health struct {
	Status string `json:"status" enum:"ok"`
}

type healthOutput struct {
	Body health
}

type status struct {
	Claim string `json:"claim" enum:"unclaimed,controller"`
}

type statusOutput struct {
	Body status
}

type report struct {
	Build     build     `json:"build"`
	DataDir   string    `json:"dataDir"`
	PID       int       `json:"pid"`
	Seed      string    `json:"seed"`
	StartedAt time.Time `json:"startedAt"`
	Listeners listeners `json:"listeners"`
}

// build is the build stamp, or only stamped false for an unstamped build.
type build struct {
	Stamped      bool   `json:"stamped"`
	Version      string `json:"version,omitempty"`
	Commit       string `json:"commit,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
	Target       string `json:"target,omitempty"`
	NativeBundle string `json:"nativeBundle,omitempty"`
	Libtorrent   string `json:"libtorrent,omitempty"`
}

type listeners struct {
	HTTP    string `json:"http"`
	Torrent string `json:"torrent"`
	Control string `json:"control"`
}

type reportOutput struct {
	Body report
}

func reportOf(f system.Facts) report {
	var b build
	if s := f.Build; s != nil {
		b = build{
			Stamped:      true,
			Version:      s.Version,
			Commit:       s.Commit,
			Fingerprint:  s.Fingerprint,
			Target:       s.Target,
			NativeBundle: s.NativeBundle,
			Libtorrent:   s.Libtorrent,
		}
	}
	return report{
		Build:     b,
		DataDir:   f.DataDir,
		PID:       f.PID,
		Seed:      f.Seed,
		StartedAt: f.StartedAt,
		Listeners: listeners{
			HTTP:    f.Listeners.HTTP.String(),
			Torrent: f.Listeners.Torrent.String(),
			Control: f.Listeners.Control,
		},
	}
}

// Routes reports facts, and the claim state from claimed, the composition
// root's one definition of a claimed Node.
func Routes(r *api.Router, facts system.Facts, claimed func(context.Context) (bool, error)) {
	o := r.Owner("system", api.Public)

	api.Register(o, api.Public, api.Op{
		ID:      "getHealth",
		Method:  http.MethodGet,
		Path:    "/health",
		Summary: "Report that the Node is serving",
		Status:  http.StatusOK,
	}, func(context.Context, identity.Anyone, *struct{}) (*healthOutput, error) {
		return &healthOutput{Body: health{Status: "ok"}}, nil
	})

	api.Register(o, api.Public, api.Op{
		ID:      "getStatus",
		Method:  http.MethodGet,
		Path:    "/status",
		Summary: "Report whether the Node is claimed",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Anyone, _ *struct{}) (*statusOutput, error) {
		done, err := claimed(ctx)
		if err != nil {
			return nil, err
		}
		if done {
			return &statusOutput{Body: status{Claim: "controller"}}, nil
		}
		return &statusOutput{Body: status{Claim: "unclaimed"}}, nil
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "getSystem",
		Method:  http.MethodGet,
		Path:    "/system",
		Summary: "Report the build, data dir, process and listeners",
		Status:  http.StatusOK,
	}, func(context.Context, identity.Admin, *struct{}) (*reportOutput, error) {
		return &reportOutput{Body: reportOf(facts)}, nil
	})
}
