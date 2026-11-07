package works

import (
	"context"
	"maps"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Settings is a Content Type's own settings, stored in its own tables and
// edited through the shared control. A type with none answers no fields
// and refuses every key with *setting.UnknownKeyError.
type Settings interface {
	SettingFields(ctx context.Context, r storage.Reader[storage.Library]) ([]setting.Field, error)
	// ApplySettings stores p, all or none, refusing a value with
	// *setting.RangeError. changed is false when every setting already
	// held what p says.
	ApplySettings(ctx context.Context, tx *storage.Tx[storage.Library], p setting.Patch) (changed bool, err error)
}

// Settings reads t's settings: its core add settings first, then the
// type's own.
func (w *Works) Settings(ctx context.Context, _ identity.Admin, t TypeID) ([]setting.Field, error) {
	reg, ok := w.types[t]
	if !ok {
		return nil, ErrNoType
	}
	var fields []setting.Field
	err := w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		fields, err = settingFields(ctx, r, t, reg)
		return err
	})
	return fields, err
}

func settingFields(ctx context.Context, r storage.Reader[file], t TypeID, reg registered) ([]setting.Field, error) {
	row, err := storedAddDefault(ctx, r, t)
	if err != nil {
		return nil, err
	}
	own, err := reg.catalog.SettingFields(ctx, r)
	if err != nil {
		return nil, err
	}
	var fields []setting.Field
	for _, s := range addSettings(t, reg.desc) {
		fields = append(fields, s.field(row))
	}
	return append(fields, own...), nil
}

// SetSettings changes t's settings in one Write, all or none. A change to
// any of the type's own publishes one works.type-changed, since every Work
// of t may now read differently. The add settings change only the next
// add.
func (w *Works) SetSettings(ctx context.Context, _ identity.Admin, t TypeID, p setting.Patch) ([]setting.Field, error) {
	reg, ok := w.types[t]
	if !ok {
		return nil, ErrNoType
	}
	core := addSettings(t, reg.desc)
	own := maps.Clone(p)
	for _, s := range core {
		delete(own, s.key)
	}
	var (
		changed bool
		fields  []setting.Field
	)
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := applyAddSettings(ctx, tx, t, core, p); err != nil {
			return err
		}
		var err error
		if changed, err = reg.catalog.ApplySettings(ctx, tx, own); err != nil {
			return err
		}
		fields, err = settingFields(ctx, tx, t, reg)
		return err
	})
	if err != nil {
		return nil, err
	}
	if changed {
		w.cfg.TypeChanged.Publish(ctx, TypeChanged{ContentType: string(t)})
	}
	return fields, nil
}

// applyAddSettings stores what p says of t's core add settings, all or
// none.
func applyAddSettings(ctx context.Context, tx *storage.Tx[file], t TypeID, core []addSetting, p setting.Patch) error {
	row, err := storedAddDefault(ctx, tx, t)
	if err != nil {
		return err
	}
	was := row
	for _, s := range core {
		if raw, ok := p[s.key]; ok {
			if err := s.apply(raw, &row); err != nil {
				return err
			}
		}
	}
	if row == was {
		return nil
	}
	return db.New(tx).PutAddDefault(ctx, db.PutAddDefaultParams(row))
}
