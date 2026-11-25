package reachability

import (
	"context"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var HealthAbout = health.About{
	ID: health.RemoteAccessID, Label: "Remote access", Phrase: "remote access",
	Events: []live.Event{ChangedEvent, nodes.ChangedEvent, mediaservers.ChangedEvent},
}

const pageHref = "/nodes/remote-access"

// HealthSource raises one warning per route whose verdict has rested on a
// failing check for Warn in Health after, and holds it unsure while the
// route's origin is away. A Not tested verdict never fails, so it never
// raises one.
func (r *Reachability) HealthSource() health.Source {
	return health.Source{Answer: r.healthAnswer}
}

func (r *Reachability) healthAnswer(ctx context.Context) (health.Answer, error) {
	var a health.Answer
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		return a, err
	}
	words := r.words(named)
	err = r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		s, _, err := settingsIn(ctx, rd)
		if err != nil {
			return err
		}
		subject, err := r.subjectsIn(ctx, rd)
		if err != nil {
			return err
		}
		results, err := resultsIn(ctx, db.New(rd))
		if err != nil {
			return err
		}
		now := r.now()
		for id, res := range results {
			if !res.read || res.failingAt.IsZero() || named[res.origin].Removed {
				continue
			}
			says := health.Says{Subject: subject[id], Text: words.Title(*res.judged.Fix()), Action: health.LinkAction(pageHref, "Open Remote access")}
			key := health.Key(id.String())
			due := res.failingAt.Add(s.warnAfter)
			switch {
			case !r.cfg.Roster.InStep(res.origin):
				says.Text = words.Skipped(res.origin, res.judgedAt.In(r.cfg.Zone).Format("15:04"))
				a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: says})
			case !now.Before(due):
				a.Items = append(a.Items, health.Item{Key: key, Level: health.Warn, Says: says})
			case a.Next.IsZero() || due.Before(a.Next):
				a.Next = due
			}
		}
		return nil
	})
	return a, err
}

// subjectsIn names each wanted Exposure.
func (r *Reachability) subjectsIn(ctx context.Context, rd storage.Reader[storage.Library]) (map[judge.ExposureID]string, error) {
	o, err := originsIn(ctx, rd, r.cfg.Roster.Self())
	if err != nil {
		return nil, err
	}
	exposures, err := wantedIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := make(map[judge.ExposureID]string, len(exposures))
	for _, e := range exposures {
		out[e.ID] = o.subject(e)
	}
	return out, nil
}

// words names Nodes by the roster's names.
func (r *Reachability) words(named map[role.NodeID]nodes.Named) Words {
	return Words{Node: func(id role.NodeID) string { return named[id].Name }}
}
