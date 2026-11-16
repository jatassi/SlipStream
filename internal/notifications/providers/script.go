package providers

// Custom Script runs a program the Admin names on a Node they pick, with
// the Webhook body as its event file. The run itself is the Node's, in
// internal/notifying, so this file follows no provider's page.

import (
	"encoding/json"
	"time"
)

var script = Provider{
	ID:    "script",
	Label: "Custom Script",
	Fields: []Field{
		{Key: "node", Label: "Node", Kind: NodeField, Help: "The Node that runs the script. It needs the Notifying Role."},
		{Key: "path", Label: "Path", Kind: Text, Help: "The program's full path on that Node."},
		{Key: "args", Label: "Arguments", Kind: List, Optional: true, Details: true, Help: "Passed to the program in this order."},
		{Key: "stop", Label: "Stop the script after", Kind: Duration, Details: true, Default: json.RawMessage("30"), Min: intp(5), Max: intp(600), Help: "A script still running then is stopped, with everything it started."},
	},
	Image:    NoImage,
	Host:     noHost,
	Build:    buildScript,
	Classify: classifyScript,
}

func buildScript(p Payload) (Outbound, error) {
	return Outbound{Run: &Script{
		Path:      p.text("path"),
		Args:      p.list("args"),
		Event:     string(p.Kind),
		EventFile: bodyOf(p),
		Stop:      time.Duration(p.number("stop")) * time.Second,
	}}, nil
}

// classifyScript reads a run: exit 0 sent, a missing path a refusal only
// the Admin can fix, and a non-zero exit or a stop a failure a retry may
// fix.
func classifyScript(r Reply, _ time.Time) Verdict {
	switch ran := r.Run; {
	case ran == nil:
		return Verdict{Outcome: Transient, Reason: r.Down}
	case ran.Missing:
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: ScriptMissing, Field: "path"}}
	case ran.TimedOut:
		return Verdict{Outcome: Transient, Reason: Reason{Code: ScriptStopped, Tail: string(ran.Tail)}}
	case ran.Exit != 0:
		return Verdict{Outcome: Transient, Reason: Reason{Code: ScriptExit, Exit: ran.Exit, Tail: string(ran.Tail)}}
	}
	return Verdict{Outcome: Sent}
}
