package indexers

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// HealthAbout follows tasks.ChangedEvent too, since the category also
// shows the Indexer tasks' warnings.
var HealthAbout = health.About{
	ID:     health.IndexersID,
	Label:  "Indexers",
	Phrase: "Indexers",
	Events: []live.Event{ChangedEvent, StatusChangedEvent, CatalogChangedEvent, tasks.ChangedEvent},
}

// HealthSource is the indexers category's items: a Warn for each Indexer
// whose catalog definition left the catalog, or whose new version doesn't
// load, while it runs the version it pins.
func (x *Indexers) HealthSource() health.Source {
	return health.Source{Answer: func(ctx context.Context) (health.Answer, error) {
		var a health.Answer
		err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			items, err := x.pinItems(ctx, db.New(r))
			a.Items = items
			return err
		})
		return a, err
	}}
}

func (x *Indexers) pinItems(ctx context.Context, q *db.Queries) ([]health.Item, error) {
	pins, err := q.ListPins(ctx)
	if err != nil {
		return nil, err
	}
	var items []health.Item
	for _, p := range pins {
		if Origin(p.Origin) != FromCatalog {
			continue
		}
		entry, err := q.GetCatalogEntry(ctx, db.DefinitionID(*p.DefinitionID))
		var text string
		switch {
		case errors.Is(err, storage.ErrNoRows):
			text = p.Name + " is no longer in the Indexer catalog."
		case err != nil:
			return nil, err
		case entry.Sha == *p.DefinitionSha:
			continue
		default:
			newer, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: entry.ID, Sha: entry.Sha})
			if err != nil {
				return nil, err
			}
			if newer.Refusal == nil {
				continue
			}
			text = "The catalog's new version of " + p.Name + " doesn't load."
		}
		items = append(items, health.Item{Key: health.KeyOf("definition", p.ID.String()), Level: health.Warn, Says: health.Says{
			Subject: p.Name,
			Text:    text + " The Indexer keeps running the version from " + x.day(p.FetchedAt.Time()) + ".",
			Action:  health.LinkAction("/settings/indexers/"+p.ID.String(), "Open "+p.Name),
		}})
	}
	return items, nil
}
