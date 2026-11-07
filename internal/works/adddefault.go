package works

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// AddDefaults are the add choices a Work of a type takes when the add
// names none: All without asides and level folders on, until remembered.
type AddDefaults struct {
	// Monitor has Asides set and New nil, so the preset decides.
	Monitor      Choice
	LevelFolders bool
}

// Remembering is the add choices an add makes its type's defaults; nil
// leaves one as it is.
type Remembering struct {
	Preset       *Preset
	Asides       *bool
	LevelFolders *bool
}

func storedAddDefault(ctx context.Context, r storage.Reader[file], t TypeID) (db.WorksAddDefault, error) {
	row, err := db.New(r).GetAddDefault(ctx, t)
	if errors.Is(err, storage.ErrNoRows) {
		return db.WorksAddDefault{ContentType: t}, nil
	}
	return row, err
}

func defaultsOf(row db.WorksAddDefault) AddDefaults {
	d := AddDefaults{Monitor: Choice{Preset: PresetAll, Asides: new(bool)}, LevelFolders: true}
	if row.Monitor != nil {
		d.Monitor.Preset = Preset(*row.Monitor)
	}
	if row.Asides != nil {
		d.Monitor.Asides = row.Asides
	}
	if row.LevelFolders != nil {
		d.LevelFolders = *row.LevelFolders
	}
	return d
}

// AddDefaultsIn reads t's add defaults inside a caller's read.
func AddDefaultsIn(ctx context.Context, r storage.Reader[file], t TypeID) (AddDefaults, error) {
	row, err := storedAddDefault(ctx, r, t)
	return defaultsOf(row), err
}

// RememberIn makes each choice m gives t's add default in tx. A choice
// that already holds stays as stored, so remembering a default keeps it a
// default.
func RememberIn(ctx context.Context, tx *storage.Tx[file], t TypeID, m Remembering) error {
	row, err := storedAddDefault(ctx, tx, t)
	if err != nil {
		return err
	}
	was, put := defaultsOf(row), row
	if p := m.Preset; p != nil && *p != was.Monitor.Preset {
		put.Monitor = new(string(*p))
	}
	if a := m.Asides; a != nil && *a != *was.Monitor.Asides {
		put.Asides = a
	}
	if f := m.LevelFolders; f != nil && *f != was.LevelFolders {
		put.LevelFolders = f
	}
	if put == row {
		return nil
	}
	return db.New(tx).PutAddDefault(ctx, db.PutAddDefaultParams(put))
}

// addSetting is one of the core's add settings for a type, kept in its
// works_add_default row.
type addSetting struct {
	key   string
	field func(db.WorksAddDefault) setting.Field
	// apply parses raw into row, refusing it with *setting.RangeError.
	apply func(raw json.RawMessage, row *db.WorksAddDefault) error
}

// addSettings are t's core add settings, which come first among its
// settings: how a new Work is monitored, a Switch for a type of one Item
// and a Choice of presets for one of many, then Include for a type that
// names an aside, then level folders for a type with levels.
func addSettings(t TypeID, d Description) []addSetting {
	out := []addSetting{monitorSetting(t, d)}
	if d.Aside.Many != "" {
		out = append(out, switchSetting(setting.Switch{Key: string(t) + ".add.asides", Label: "Include " + strings.ToLower(d.Aside.Many), Default: false},
			func(row *db.WorksAddDefault) **bool { return &row.Asides }))
	}
	if len(d.Levels) > 0 {
		out = append(out, switchSetting(setting.Switch{Key: string(t) + ".add.levelFolders", Label: d.Levels[0].One + " folders", Default: true},
			func(row *db.WorksAddDefault) **bool { return &row.LevelFolders }))
	}
	return out
}

func switchSetting(s setting.Switch, of func(*db.WorksAddDefault) **bool) addSetting {
	return addSetting{
		key:   s.Key,
		field: func(row db.WorksAddDefault) setting.Field { return s.Field(*of(&row)) },
		apply: func(raw json.RawMessage, row *db.WorksAddDefault) error {
			on, err := s.ParseJSON(raw)
			*of(row) = on.Override()
			return err
		},
	}
}

func monitorSetting(t TypeID, d Description) addSetting {
	key := string(t) + ".add.monitor"
	if d.Items == ItemsOne {
		s := setting.Switch{Key: key, Label: "New " + strings.ToLower(d.Work.Many) + " are monitored", Default: true}
		return addSetting{
			key: key,
			field: func(row db.WorksAddDefault) setting.Field {
				if row.Monitor == nil {
					return s.Field(nil)
				}
				return s.Field(new(Preset(*row.Monitor) == PresetAll))
			},
			apply: func(raw json.RawMessage, row *db.WorksAddDefault) error {
				on, err := s.ParseJSON(raw)
				row.Monitor = nil
				if v := on.Override(); v != nil {
					row.Monitor = new(string(map[bool]Preset{true: PresetAll, false: PresetNone}[*v]))
				}
				return err
			},
		}
	}
	c := setting.Choice[Preset]{Key: key, Label: d.Item.Many + " to monitor", Default: PresetAll}
	for _, p := range presets {
		c.Options = append(c.Options, setting.Option[Preset]{Value: p, Label: presetLabel(p, d)})
	}
	return addSetting{
		key: key,
		field: func(row db.WorksAddDefault) setting.Field {
			return c.Field((*Preset)(row.Monitor))
		},
		apply: func(raw json.RawMessage, row *db.WorksAddDefault) error {
			p, err := c.ParseJSON(raw)
			row.Monitor = (*string)(p.Override())
			return err
		},
	}
}

// presetLabel names p for d's Works, from its outermost level's noun.
func presetLabel(p Preset, d Description) string {
	level := d.Work.One
	if len(d.Levels) > 0 {
		level = d.Levels[0].One
	}
	return map[Preset]string{
		PresetAll: "All", PresetFuture: "Future", PresetMissing: "Missing", PresetExisting: "Existing",
		PresetFirst: "First " + strings.ToLower(level), PresetLatest: "Latest " + strings.ToLower(level), PresetNone: "None",
	}[p]
}
