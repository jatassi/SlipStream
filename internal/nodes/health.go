package nodes

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var HealthAbout = health.About{
	ID:     health.NodesID,
	Label:  "Nodes",
	Phrase: "Nodes",
	Events: []live.Event{ChangedEvent},
}

// HealthSource answers what is wrong with the Nodes from the board and the
// probe verdicts kept from their facts, and re-runs a Node's probes for
// Check now. It stores nothing.
func (s *Service) HealthSource() health.Source {
	return health.Source{
		Answer: func(context.Context) (health.Answer, error) {
			s.mu.Lock()
			probes := maps.Clone(s.probes)
			s.mu.Unlock()
			return itemsOf(s.Board(), probes), nil
		},
		Check: s.healthCheck,
	}
}

// probeFixes is what to do about each reason a probe finds.
var probeFixes = map[role.ProbeReason]string{
	role.EngineNotListening:      "Check that no other program holds its torrent port, then Check now.",
	role.DownloadsNotWritable:    "Let SlipStream write to its downloads folder, then Check now.",
	role.DownloadsOnNetworkShare: "Move its downloads folder to a local disk, then Check now.",
	role.DataNotWritable:         "Let SlipStream write to its data folder, then Check now.",
	role.HTTPNotListening:        "Check that no other program holds its web port, then Check now.",
}

// itemsOf is the Nodes category's answer on b: a Node lapsed or on another
// version fails; a probe that fails or is weak for a Role the Node holds
// fails or warns; the Controller covering a Role warns. A Node away inside
// its grace, and a probe whose verdict isn't in since the Controller
// started, is unsure, so a blip or a restart neither raises nor clears.
func itemsOf(b *Board, probes map[ID][]role.ProbeResult) health.Answer {
	var a health.Answer
	for i, v := range b.nodes {
		key := health.KeyOf(v.ID.String())
		switch v.Standing {
		case InStep:
		case Reconnecting, Offline:
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{Subject: v.Name, Text: awayWords(v)}})
		case LapsedNode:
			a.Items = append(a.Items, nodeItem(v, v.Name+" has been offline past the offline grace. Start SlipStream on it, or remove it if it is gone.", "Offline since"))
		case MismatchedNode:
			a.Items = append(a.Items, nodeItem(v, v.Name+" runs another version of SlipStream. Update it to the Controller's version.", "Since"))
		}
		for _, p := range role.Probes() {
			roles := probing(b, i, p)
			if len(roles) == 0 {
				continue
			}
			pk := health.KeyOf(v.ID.String(), string(p))
			subject := labels(roles) + " on " + v.Name
			verdict, ok := verdictOf(probes[v.ID], p)
			switch {
			case v.Standing != InStep:
				a.Unsure = append(a.Unsure, health.Unsure{Key: pk, Says: health.Says{Subject: subject, Text: awayWords(v) + " Its probes run again once it is back."}})
			case !ok:
				a.Unsure = append(a.Unsure, health.Unsure{Key: pk, Says: health.Says{Subject: subject, Text: v.Name + " hasn't reported its probes since SlipStream started."}})
			case verdict.Reason == role.EngineNotListening:
				// Download Clients reports a listener that can't bind, beside the
				// port it is set to.
			case verdict.Fit == role.Fails:
				a.Items = append(a.Items, probeItem(pk, health.Fail, subject, fmt.Sprintf("%s can't run on %s.", labels(roles), v.Name), verdict))
			case verdict.Fit == role.Weak:
				a.Items = append(a.Items, probeItem(pk, health.Warn, subject, fmt.Sprintf("%s may run poorly on %s.", labels(roles), v.Name), verdict))
			}
		}
	}
	a.Items = append(a.Items, coveringItems(b)...)
	return a
}

func nodeItem(v NodeView, text, since string) health.Item {
	return health.Item{Key: health.KeyOf(v.ID.String()), Level: health.Fail, Says: health.Says{
		Subject: v.Name,
		Text:    text,
		Detail:  []health.Fact{{Label: since, At: v.Since}},
		Action:  health.LinkAction("/nodes/"+v.ID.String(), "Open "+v.Name),
	}}
}

func probeItem(key health.Key, level health.Level, subject, verdict string, p role.ProbeResult) health.Item {
	return health.Item{Key: key, Level: level, Says: health.Says{
		Subject: subject,
		Text:    verdict + " " + p.Reason.Words() + " " + probeFixes[p.Reason],
		Detail:  []health.Fact{{Label: "Probed", At: p.At}},
		Action:  health.CheckAction(),
	}}
}

// coveringItems is a warning for each Role the Controller covers, naming
// the holders it covers for.
func coveringItems(b *Board) []health.Item {
	var controller NodeView
	for _, v := range b.nodes {
		if v.Controller {
			controller = v
		}
	}
	var out []health.Item
	for _, r := range b.covering().Names() {
		var holders []NodeView
		for _, v := range b.nodes {
			if !v.Controller && b.Who(r).Holds(v.ID) {
				holders = append(holders, v)
			}
		}
		names := make([]string, len(holders))
		for i, h := range holders {
			names[i] = h.Name
		}
		out = append(out, health.Item{Key: health.KeyOf("covering", string(r)), Level: health.Warn, Says: health.Says{
			Subject: r.Label() + " on " + controller.Name,
			Text: fmt.Sprintf("%s %s in place of %s. Bring %s back, or give %s to another Node.",
				controller.Name, r.Purpose(), and(names), and(names), r.Label()),
			Action: health.LinkAction("/nodes/"+holders[0].ID.String(), "Open "+holders[0].Name),
		}})
	}
	return out
}

// probing is the Roles the Node at i holds that p judges, in table order.
func probing(b *Board, i int, p role.Probe) []role.Name {
	var out []role.Name
	for _, r := range role.Names() {
		if r.Rule().Probe == p && b.Who(r).Holds(b.nodes[i].ID) {
			out = append(out, r)
		}
	}
	return out
}

func verdictOf(kept []role.ProbeResult, p role.Probe) (role.ProbeResult, bool) {
	for _, k := range kept {
		if k.Probe == p {
			return k, true
		}
	}
	return role.ProbeResult{}, false
}

// awayWords is why a Node that isn't in step can't be judged or reached.
func awayWords(v NodeView) string {
	switch v.Standing {
	case Reconnecting:
		return v.Name + " is reconnecting."
	case MismatchedNode:
		return v.Name + " runs another version of SlipStream."
	case InStep, Offline, LapsedNode:
	}
	return v.Name + " is offline."
}

// healthCheck re-runs the probes of the Node item names, or of every Node
// when item is "". A Node out of step is unreached.
func (s *Service) healthCheck(ctx context.Context, item health.Key) ([]health.Unreached, error) {
	b := s.Board()
	views := b.nodes
	if item != "" {
		head, _, _ := strings.Cut(string(item), ".")
		u, err := storage.ParseUUID(head)
		if err != nil {
			return nil, fmt.Errorf("item %s names no Node", item)
		}
		v, ok := b.Node(ID{UUID: u})
		if !ok {
			return nil, nil
		}
		views = []NodeView{v}
	}
	var (
		mu        sync.Mutex
		unreached []health.Unreached
		errs      []error
		wg        sync.WaitGroup
	)
	for _, v := range views {
		wg.Go(func() {
			words, err := s.reprobe(ctx, v)
			mu.Lock()
			defer mu.Unlock()
			if words != "" {
				unreached = append(unreached, health.Unreached{Key: item, Words: words})
			}
			if err != nil {
				errs = append(errs, err)
			}
		})
	}
	wg.Wait()
	slices.SortFunc(unreached, func(a, b health.Unreached) int { return cmp.Compare(a.Words, b.Words) })
	return unreached, errors.Join(errs...)
}

// reprobe re-runs v's probes, answering why it couldn't instead when v is
// out of step.
func (s *Service) reprobe(ctx context.Context, v NodeView) (unreached string, err error) {
	if v.Standing != InStep {
		return awayWords(v), nil
	}
	err = s.Probe(ctx, v.ID)
	switch {
	case errors.Is(err, nodesystem.ErrOffline):
		return v.Name + " is offline.", nil
	case errors.Is(err, nodesystem.ErrMismatched):
		return v.Name + " runs another version of SlipStream.", nil
	case err != nil:
		return "", fmt.Errorf("probe %s: %w", v.Name, err)
	}
	return "", nil
}
