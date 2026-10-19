package role

import (
	"maps"
	"slices"
	"time"

	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
)

func (f Fit) String() string { return [...]string{"", "fits", "weak", "fails"}[f] }

// ProbeReason is why a probe found its Node weak or failing.
type ProbeReason string

const (
	EngineNotListening      ProbeReason = "engine_not_listening"
	DownloadsNotWritable    ProbeReason = "downloads_not_writable"
	DownloadsOnNetworkShare ProbeReason = "downloads_on_network_share"
	DataNotWritable         ProbeReason = "data_not_writable"
	HTTPNotListening        ProbeReason = "http_not_listening"
)

// probeReasons is the Probe that finds each ProbeReason.
var probeReasons = map[ProbeReason]Probe{
	EngineNotListening:      EngineStarts,
	DownloadsNotWritable:    EngineStarts,
	DownloadsOnNetworkShare: EngineStarts,
	DataNotWritable:         DataWritable,
	HTTPNotListening:        ListenerUp,
}

// ProbeReasons is every ProbeReason, sorted.
func ProbeReasons() []ProbeReason { return slices.Sorted(maps.Keys(probeReasons)) }

var probeWords = map[ProbeReason]string{
	EngineNotListening:      "Its torrent engine isn't listening for peers.",
	DownloadsNotWritable:    "Its downloads folder can't be written to.",
	DownloadsOnNetworkShare: "Its downloads folder is on a network share.",
	DataNotWritable:         "Its data folder can't be written to.",
	HTTPNotListening:        "Its web server isn't listening.",
}

// Words is what r found, as a sentence.
func (r ProbeReason) Words() string { return probeWords[r] }

// Probe is the probe that finds r.
func (r ProbeReason) Probe() Probe { return probeReasons[r] }

// ProbeResult is a probe's verdict on the Node that ran it.
type ProbeResult struct {
	Probe Probe
	Fit   Fit
	// Reason is why it is weak or fails, "" when it fits.
	Reason ProbeReason
	At     time.Time
}

// ProbeFacts is results as the wire carries them.
func ProbeFacts(results []ProbeResult) []*nodev1.ProbeFact {
	out := make([]*nodev1.ProbeFact, len(results))
	for i, p := range results {
		out[i] = &nodev1.ProbeFact{Probe: string(p.Probe), Fit: int32(p.Fit), Reason: string(p.Reason), AtUnixMs: p.At.UnixMilli()}
	}
	return out
}

// ProbesOf is the probe verdicts facts carry.
func ProbesOf(facts []*nodev1.ProbeFact) []ProbeResult {
	var out []ProbeResult
	for _, p := range facts {
		out = append(out, ProbeResult{
			Probe: Probe(p.GetProbe()), Fit: Fit(p.GetFit()), //nolint:gosec // a Fit the Node's own build sent
			Reason: ProbeReason(p.GetReason()), At: time.UnixMilli(p.GetAtUnixMs()),
		})
	}
	return out
}
