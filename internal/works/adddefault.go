package works

import (
	"context"
	"errors"
	"strings"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// addMonitorDefault is add.monitor while a type stores none.
const addMonitorDefault = true

// addMonitor is t's <type>.add.monitor: whether a Work added with no
// Monitor choice has its Items monitored.
func addMonitor(t TypeID, d Description) setting.Switch {
	return setting.Switch{Key: string(t) + ".add.monitor", Label: "New " + strings.ToLower(d.Work.Many) + " are monitored", Default: addMonitorDefault}
}

func storedAddMonitor(ctx context.Context, r storage.Reader[file], t TypeID) (*bool, error) {
	row, err := db.New(r).GetAddDefault(ctx, t)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	case row.Monitor == nil:
		return nil, nil
	}
	on := Preset(*row.Monitor) == PresetAll
	return &on, nil
}

func storeAddMonitor(ctx context.Context, tx *storage.Tx[file], t TypeID, on *bool) error {
	q := db.New(tx)
	row, err := q.GetAddDefault(ctx, t)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	put := db.PutAddDefaultParams{ContentType: t, Asides: row.Asides, LevelFolders: row.LevelFolders}
	if on != nil {
		preset := string(PresetNone)
		if *on {
			preset = string(PresetAll)
		}
		put.Monitor = &preset
	}
	return q.PutAddDefault(ctx, put)
}

// AddMonitorIn is whether a Work of t added with no Monitor choice is
// monitored.
func AddMonitorIn(ctx context.Context, r storage.Reader[file], t TypeID) (bool, error) {
	on, err := storedAddMonitor(ctx, r, t)
	if err != nil || on == nil {
		return addMonitorDefault, err
	}
	return *on, nil
}

// SetAddMonitorIn makes on t's add.monitor in tx, writing only when it
// differs. changed is whether it did.
func SetAddMonitorIn(ctx context.Context, tx *storage.Tx[file], t TypeID, on bool) (changed bool, err error) {
	was, err := AddMonitorIn(ctx, tx, t)
	if err != nil || was == on {
		return false, err
	}
	return true, storeAddMonitor(ctx, tx, t, &on)
}
