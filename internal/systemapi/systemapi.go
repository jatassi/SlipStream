// Package systemapi holds the Node's liveness, claim state and runtime
// facts, and each Node's system as its Controller manages it: its facts,
// settings, restart, log stream and log archive.
package systemapi

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/system"
)

type health struct {
	Status string `json:"status" enum:"ok"`
}

type healthOutput struct {
	Body health
}

type status struct {
	Claim    string   `json:"claim" enum:"unclaimed,controller,worker"`
	Passkeys passkeys `json:"passkeys" doc:"Whether the address this request reached SlipStream at can hold a passkey"`
	Removed  *removed `json:"removed,omitempty" doc:"Set on an Unclaimed Node its Controller removed, until it joins again"`
}

type removed struct {
	At         time.Time `json:"at"`
	Controller string    `json:"controller" doc:"The name of the Controller that removed the Node"`
}

type passkeys struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty" enum:"ip_address,plain_http,not_a_domain" doc:"Why it can't: an IP address, a name over plain HTTP other than localhost, or a host that is not a domain name"`
}

type statusOutput struct {
	Body status
}

type report struct {
	Build     build     `json:"build"`
	DataDir   string    `json:"dataDir"`
	LogsDir   string    `json:"logsDir"`
	PID       int       `json:"pid"`
	Seed      string    `json:"seed"`
	StartedAt time.Time `json:"startedAt"`
	HTTP      httpFacts `json:"http"`
	Listeners listeners `json:"listeners"`
}

type httpFacts struct {
	Configured string `json:"configured" doc:"http.listen at start; listeners.http differs after a fallback to a later port"`
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
	Link    string `json:"link"`
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
		LogsDir:   f.LogsDir,
		PID:       f.PID,
		Seed:      f.Seed,
		StartedAt: f.StartedAt,
		HTTP:      httpFacts{Configured: f.HTTPConfigured.String()},
		Listeners: listeners{
			HTTP:    f.Listeners.HTTP.String(),
			Torrent: f.Listeners.Torrent.String(),
			Link:    f.Listeners.Link.String(),
			Control: f.Listeners.Control,
		},
	}
}

// Claims is the composition root's one definition of the Node's claim.
type Claims interface {
	Claim(context.Context) (system.Claim, error)
}

// Node is the running Node as the operations reach it.
type Node interface {
	Claims
	Systems
	// Removal is when and by whom the Node was removed, false unless its
	// Controller removed it since it last joined.
	Removal() (pairing.Removal, bool)
}

// Routes reports this Node's facts and claim, and every Node's system.
func Routes(r *api.Router, facts system.Facts, node Node) {
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
		Summary: "Report whether the Node is Unclaimed, a Controller or a Worker, and whether this address can hold a passkey",
		Status:  http.StatusOK,
	}, func(ctx context.Context, a identity.Anyone, _ *struct{}) (*statusOutput, error) {
		claim, err := node.Claim(ctx)
		if err != nil {
			return nil, err
		}
		v := a.Passkeys()
		out := status{Claim: claim.String(), Passkeys: passkeys{Available: v.Available, Reason: string(v.Reason)}}
		if r, ok := node.Removal(); ok && claim == system.Unclaimed {
			out.Removed = &removed{At: r.At, Controller: r.ControllerName}
		}
		return &statusOutput{Body: out}, nil
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

	h := nodeHandlers{node}

	api.Register(o, api.Admin, api.Op{
		ID:      "getNodeSystem",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/system",
		Summary: "Report a Node's build, data dir, process and listeners",
		Status:  http.StatusOK,
		Errors:  reach,
	}, h.getNodeSystem)

	api.Register(o, api.Admin, api.Op{
		ID:      "getNodeConfig",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/config",
		Summary: "Get a Node's settings, by section",
		Status:  http.StatusOK,
		Errors:  reach,
	}, h.getNodeConfig)

	api.Register(o, api.Admin, api.Op{
		ID:      "updateNodeConfig",
		Method:  http.MethodPatch,
		Path:    "/nodes/{id}/config",
		Summary: "Change a Node's settings, every key or none",
		Status:  http.StatusOK,
		Errors:  slices.Concat(reach, []*api.Code{setByEnvironment, settingapi.ErrOutOfRange, portUsedByNode, settingapi.ErrUnknown, configNotWritable}),
	}, h.updateNodeConfig)

	api.Register(o, api.Admin, api.Op{
		ID:      "restartNode",
		Method:  http.MethodPost,
		Path:    "/nodes/{id}/restart",
		Summary: "Restart a Node. The response comes before it stops",
		Status:  http.StatusAccepted,
		Errors:  slices.Concat(reach, []*api.Code{alreadyRestarting}),
	}, h.restartNode)

	api.RegisterStream(o, api.Admin, api.Op{
		ID:      "streamNodeLogs",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/logs/stream",
		Summary: "Stream a Node's log: its recent records, then each as it comes",
		Status:  http.StatusOK,
		Errors:  reach,
	}, logEvents, h.streamNodeLogs)

	api.RegisterDownload(o, api.Admin, api.Op{
		ID:      "downloadNodeLogs",
		Method:  http.MethodGet,
		Path:    "/nodes/{id}/logs/archive",
		Summary: "Download a zip of a Node's log folder",
		Status:  http.StatusOK,
		Errors:  reach,
	}, "application/zip", h.downloadNodeLogs)
}
