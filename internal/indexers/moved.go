package indexers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/indexerfields"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/storage"
)

// moveMoved runs migrateMoved in its own Write. Run calls it once the
// library key is open, since a move opens and seals settings.
func (x *Indexers) moveMoved(ctx context.Context) error {
	var moved []ID
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		moved, err = x.migrateMoved(ctx, tx, storage.Now())
		return err
	})
	if err != nil {
		return fmt.Errorf("move the Indexers of ported sites the catalog took over: %w", err)
	}
	for _, id := range moved {
		x.changed(ctx, id)
	}
	return nil
}

// migrateMoved makes each native Indexer on a port whose site moved to
// the catalog a catalog Indexer on that definition, inside tx: the same
// Indexer at its next revision, with its settings mapped by the port's
// field map, its Tracker keyed by the definition's id too and a note for
// the Indexer page. Its history stays. An Indexer whose definition the
// catalog doesn't hold at a version that loads moves at a later refresh,
// and one already moved is a catalog Indexer, so running it again moves
// nothing. It answers the Indexers it moved.
func (x *Indexers) migrateMoved(ctx context.Context, tx *storage.Tx[file], now storage.Time) ([]ID, error) {
	moves := x.cfg.Ports.Moved()
	if len(moves) == 0 {
		return nil, nil
	}
	all, err := All(ctx, tx)
	if err != nil {
		return nil, err
	}
	var moved []ID
	for _, m := range moves {
		on := slices.DeleteFunc(slices.Clone(all), func(idx Indexer) bool { p, ok := idx.Source.(Port); return !ok || p.ID != m.ID })
		if len(on) == 0 {
			continue
		}
		def, err := x.movedTo(ctx, tx, m)
		if errors.Is(err, errNotYet) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, idx := range on {
			if err := x.moveTo(ctx, tx, idx, m, def, now); err != nil {
				return nil, err
			}
			moved = append(moved, idx.ID)
		}
	}
	return moved, nil
}

// errNotYet is a moved port's definition the catalog doesn't hold at a
// version that loads.
var errNotYet = errors.New("the catalog holds no loading version of the definition")

// target is the definition version a moved port's Indexers move to.
type target struct {
	id, sha string
	fields  []indexerfields.Field
	links   []string
}

func (x *Indexers) movedTo(ctx context.Context, tx *storage.Tx[file], m port.Moved) (target, error) {
	q := db.New(tx)
	entry, err := q.GetCatalogEntry(ctx, db.DefinitionID(m.Definition))
	if errors.Is(err, storage.ErrNoRows) {
		return target{}, errNotYet
	}
	if err != nil {
		return target{}, err
	}
	row, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: entry.ID, Sha: entry.Sha})
	if err != nil {
		return target{}, err
	}
	s, loads, err := summaryOf(row)
	switch {
	case err != nil:
		return target{}, err
	case !loads || len(s.Links) == 0:
		return target{}, errNotYet
	}
	fields := slices.DeleteFunc(slices.Clone(s.Fields), func(f indexerfields.Field) bool { return !settable(s, f) })
	return target{id: m.Definition, sha: entry.Sha, fields: fields, links: s.Links}, nil
}

func (x *Indexers) moveTo(ctx context.Context, tx *storage.Tx[file], idx Indexer, m port.Moved, def target, now storage.Time) error {
	p, _ := idx.Source.(Port)
	secrets, err := x.open(idx)
	if err != nil {
		return err
	}
	byDefinition := map[string]string{}
	for from, to := range m.Fields {
		if v, ok := p.Plain[from]; ok {
			byDefinition[to] = v
		} else if v, ok := secrets.Fields[from]; ok {
			byDefinition[to] = v
		}
	}
	plain, sealed := map[string]string{}, map[string]string{}
	for _, f := range def.fields {
		v, ok := byDefinition[f.Name]
		if !ok || indexerfields.Fits(f, v) != nil {
			continue
		}
		if indexerfields.Sealed(f) {
			sealed[f.Name] = v
		} else {
			plain[f.Name] = v
		}
	}
	address := p.Address
	if !slices.Contains(def.links, address) {
		address = def.links[0]
	}
	secrets.Fields = sealed
	sealedSecrets, err := x.seal(idx.ID, secrets)
	if err != nil {
		return err
	}
	plainJSON, err := json.Marshal(plain)
	if err != nil {
		return fmt.Errorf("encode the plain settings: %w", err)
	}
	notes, err := json.Marshal([]string{idx.Name + " now runs from its catalog definition. It kept its settings, Tracker and history."})
	if err != nil {
		return fmt.Errorf("encode the notes: %w", err)
	}
	q := db.New(tx)
	_, err = q.MoveToDefinition(ctx, db.MoveToDefinitionParams{
		DefinitionID: &def.id, DefinitionSha: &def.sha, BaseAddress: &address, Plain: string(plainJSON), SecretsSealed: sealedSecrets,
		Notes: string(notes), UpdatedAt: now, ID: idx.ID,
	})
	if err != nil {
		return err
	}
	tracker, err := q.GetTrackerOfKey(ctx, "definition:"+string(m.ID))
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return q.AddTrackerKey(ctx, db.AddTrackerKeyParams{Key: "definition:" + m.Definition, TrackerID: tracker})
}
