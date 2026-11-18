package subtitles

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// HealthAbout follows nodes.ChangedEvent for the Subtitles Role's
// holders, targets.ChangedEvent for Wanted Languages, and tasks for
// Subtitle search's warnings.
var HealthAbout = health.About{
	ID:     health.SubtitlesID,
	Label:  "Subtitles",
	Phrase: "subtitles",
	Events: []live.Event{ChangedEvent, ProvidersChangedEvent, nodes.ChangedEvent, targets.ChangedEvent, tasks.ChangedEvent},
}

var openSubtitles = health.LinkAction("/settings/subtitles", "Open Subtitles")

// HealthSource is the Subtitles category's items: a Fail for each provider
// that rejected its key, a Warn for each one failing past
// "Warn when a Subtitle Provider keeps failing for", a Fail while a Profile
// has a Wanted Language and no Node has the Subtitles Role, a Warn for each
// Content Type with Wanted Languages that no provider that is on searches,
// and a Fail for each Node whose audio scans fail for the Node itself.
// Next is the soonest failing provider's warning edge.
func (s *Subtitles) HealthSource() health.Source {
	return health.Source{Answer: func(ctx context.Context) (health.Answer, error) {
		var a health.Answer
		err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			a, err = s.healthAnswer(ctx, r, s.cfg.Clock())
			return err
		})
		return a, err
	}}
}

func (s *Subtitles) healthAnswer(ctx context.Context, r storage.Reader[storage.Library], now time.Time) (health.Answer, error) {
	q := db.New(r)
	row, err := q.GetSettings(ctx)
	if err != nil {
		return health.Answer{}, err
	}
	providers, err := s.providersIn(ctx, r)
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	warn := WarnAfter.Value(row.ProviderWarnAfterSeconds)
	for _, p := range providers {
		s.judgeProvider(&a, p, warn, now)
	}
	wanted, err := q.ListWantedTypes(ctx)
	if err != nil {
		return health.Answer{}, err
	}
	if len(wanted) > 0 && len(s.cfg.Roster.Holding(role.Subtitles)) == 0 {
		a.Items = append(a.Items, health.Item{Key: health.KeyOf("no-holder"), Level: health.Fail, Says: health.Says{
			Subject: "Subtitles Role",
			Text:    "No Node has the Subtitles Role, so SlipStream can't download or time subtitles. Assign it on the Nodes page.",
			Action:  health.LinkAction("/nodes", "Open Nodes"),
		}})
	}
	for _, t := range wanted {
		if it, ok := s.uncovered(works.TypeID(t), providers); ok {
			a.Items = append(a.Items, it)
		}
	}
	faults, err := q.ListNodeFaults(ctx)
	for _, f := range faults {
		a.Items = append(a.Items, s.faultItem(f))
	}
	return a, err
}

func (s *Subtitles) judgeProvider(a *health.Answer, p Provider, warn time.Duration, now time.Time) {
	switch {
	case !p.On:
	case p.Standing(now) == Rejected:
		a.Items = append(a.Items, health.Item{Key: health.KeyOf("key-rejected", string(p.Spec.ID)), Level: health.Fail, Says: health.Says{
			Subject: p.Spec.Label,
			Text:    p.Spec.Label + " rejected your key and responded \"" + providerWords(p.Paused.Words) + "\". Add a new key in Settings > Subtitles.",
			Action:  openSubtitles,
		}})
	case p.FailingSince.IsZero():
	case now.Before(p.FailingSince.Add(warn)):
		if edge := p.FailingSince.Add(warn); a.Next.IsZero() || edge.Before(a.Next) {
			a.Next = edge
		}
	default:
		detail := []health.Fact{{Label: "Failing since", At: p.FailingSince}}
		if p.LastError != "" {
			detail = append(detail, health.Fact{Label: "Last error", Text: p.LastError})
		}
		a.Items = append(a.Items, health.Item{Key: health.KeyOf("failing", string(p.Spec.ID)), Level: health.Warn, Says: health.Says{
			Subject: p.Spec.Label,
			Text:    p.Spec.Label + " isn't responding. SlipStream retries automatically.",
			Detail:  detail,
			Action:  openSubtitles,
		}})
	}
}

// uncovered is the Warn of typ, whose Profiles list Wanted Languages, when
// no provider that can search covers its kind.
func (s *Subtitles) uncovered(typ works.TypeID, providers []Provider) (health.Item, bool) {
	facet, ok := s.cfg.Types[typ]
	if !ok {
		return health.Item{}, false
	}
	k := kindOfType(facet)
	covers := func(p Provider) bool { return slices.Contains(p.Spec.Kinds, k) }
	if k == 0 || slices.ContainsFunc(providers, func(p Provider) bool { return p.searches() && covers(p) }) {
		return health.Item{}, false
	}
	d, _ := s.cfg.Describe(typ)
	var off, keyed []string
	for _, p := range providers {
		switch {
		case !covers(p):
		case p.Spec.Keyed && !p.HasKey:
			keyed = append(keyed, p.Spec.Label)
		default:
			off = append(off, p.Spec.Label)
		}
	}
	fix := "Add a " + strings.Join(keyed, " or ") + " key."
	if len(off) > 0 {
		fix = "Turn on " + strings.Join(off, " or ") + " in Settings > Subtitles."
	}
	return health.Item{Key: health.KeyOf("uncovered", string(typ)), Level: health.Warn, Says: health.Says{
		Subject: d.Work.Many,
		Text:    "No Subtitle Provider that is on searches " + strings.ToLower(d.Work.Many) + ". " + fix,
		Action:  openSubtitles,
	}}, true
}

func (s *Subtitles) faultItem(f db.ListNodeFaultsRow) health.Item {
	name := s.cfg.Roster.Name(f.NodeID)
	return health.Item{Key: health.KeyOf("map-fault", f.NodeID.String()), Level: health.Fail, Says: health.Says{
		Subject: name,
		Text:    name + " can't scan audio because its bundled ffmpeg doesn't start. Restart " + name + ", and reinstall SlipStream on it if that doesn't fix it.",
		Detail:  []health.Fact{{Label: "Failing since", At: f.FaultAt.Time()}, {Label: "ffmpeg", Text: deref(f.FaultWords)}},
		Action:  health.LinkAction("/nodes/"+f.NodeID.String(), "Open "+name),
	}}
}
