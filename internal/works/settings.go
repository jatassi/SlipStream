package works

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
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

// Settings reads t's own settings.
func (w *Works) Settings(ctx context.Context, _ identity.Admin, t TypeID) ([]setting.Field, error) {
	reg, ok := w.types[t]
	if !ok {
		return nil, ErrNoType
	}
	var fields []setting.Field
	err := w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		fields, err = reg.catalog.SettingFields(ctx, r)
		return err
	})
	return fields, err
}

// SetSettings changes t's own settings in one Write and, when that changed
// any, publishes one works.type-changed, since every Work of t may now
// read differently.
func (w *Works) SetSettings(ctx context.Context, _ identity.Admin, t TypeID, p setting.Patch) ([]setting.Field, error) {
	reg, ok := w.types[t]
	if !ok {
		return nil, ErrNoType
	}
	var (
		changed bool
		fields  []setting.Field
	)
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if changed, err = reg.catalog.ApplySettings(ctx, tx, p); err != nil {
			return err
		}
		fields, err = reg.catalog.SettingFields(ctx, tx)
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
