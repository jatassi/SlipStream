package imports

import (
	"context"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	// StopShare is the guard: a comparison that would find more than this
	// share of a root's files gone marks none. 100 never stops on the
	// share.
	StopShare = setting.Count{
		Key: "library.scan.stopShare", Label: "Stop a scan when this share of a root's files is gone",
		Default: 25, Min: 1, Max: 100, Unit: setting.Noun{One: "%", Many: "%"},
	}
	Probes = setting.Count{
		Key: "library.scan.probes", Label: "Files each Node reads media info from at once",
		Default: 2, Min: 1, Max: 8, Unit: setting.Noun{One: "file", Many: "files"},
	}
)

// Settings is Library scan's settings an Admin sets in Settings > Library
// scan: the stop share and how many files a Node probes at once.
func (im *Imports) Settings() setting.Panel { return panel{im} }

type panel struct{ im *Imports }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	s, err := db.New(p.im.cfg.Library).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return []setting.Field{StopShare.Field(s.StopShare), Probes.Field(s.Probes)}, nil
}

// Apply stores the patch, all or none.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	for key := range patch {
		if key != StopShare.Key && key != Probes.Key {
			return &setting.UnknownKeyError{Key: key}
		}
	}
	return p.im.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		s, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		for _, c := range []struct {
			d  setting.Count
			at **int64
		}{{StopShare, &s.StopShare}, {Probes, &s.Probes}} {
			raw, ok := patch[c.d.Key]
			if !ok {
				continue
			}
			v, err := c.d.ParseJSON(raw)
			if err != nil {
				return err
			}
			*c.at = v.Override()
		}
		return q.SetSettings(ctx, db.SetSettingsParams{StopShare: s.StopShare, Probes: s.Probes})
	})
}
