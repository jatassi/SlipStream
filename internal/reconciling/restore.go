package reconciling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// WroteRestore is whether the restore id already wrote r's library: its Write
// runs once, so a start stopped after it commits never runs it again.
func WroteRestore(ctx context.Context, r storage.Reader[file], id string) (bool, error) {
	state, err := db.New(r).GetState(ctx)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read the Reconciling state: %w", err)
	}
	return string(state.RestoreID) == id, nil
}

type RestoredFacts struct {
	Backup  string    `json:"backup"`
	TakenAt time.Time `json:"takenAt"`
}

func (RestoredFacts) About() entry.About { return entry.About{} }

type ReplacedFacts struct {
	Controller string `json:"controller"`
}

func (ReplacedFacts) About() entry.About { return entry.About{} }

type FinishedFacts struct{}

func (FinishedFacts) About() entry.About { return entry.About{} }

// RestoredRow is Backup restored, and ReplacedRow Controller replaced, on
// a new machine only. Both are recorded in the first Write after the
// History catalog seals, since the restore's Write runs before it.
var RestoredRow = entry.Row[RestoredFacts]{
	Spec: entry.Spec{Kind: "backup_restored", Label: "Backup restored", Group: entry.Backups, OnByDefault: true, BellByDefault: true},
	Render: func(f RestoredFacts, c entry.Context) entry.Message {
		return entry.Message{
			Title: "Backup restored",
			Lines: []string{fmt.Sprintf("SlipStream restored the Backup from %s, %s. It's matching the restored library with the torrents and files on your Nodes.", c.Day(f.TakenAt), c.Clock(f.TakenAt))},
			Link:  entry.Link{Path: "/reconciling"},
			Tone:  entry.Good,
		}
	},
}

var ReplacedRow = entry.Row[ReplacedFacts]{
	Spec: entry.Spec{Kind: "controller_replaced", Label: "Controller replaced", Group: entry.Backups, OnByDefault: true, BellByDefault: true},
	Render: func(f ReplacedFacts, _ entry.Context) entry.Message {
		return entry.Message{
			Title: "Controller replaced",
			Lines: []string{"This machine is now the Controller " + f.Controller + ". Make sure the old machine stays off. Each Worker you pair again pauses it when it next connects to it."},
			Link:  entry.Link{Path: "/reconciling"},
			Tone:  entry.Info,
		}
	},
}

// FinishedRow is Reconciling finished, recorded in the Write that ends it.
var FinishedRow = entry.Row[FinishedFacts]{
	Spec: entry.Spec{Kind: "reconciling_finished", Label: "Reconciling finished", Group: entry.Backups, OnByDefault: true, BellByDefault: true},
	Render: func(FinishedFacts, entry.Context) entry.Message {
		return entry.Message{
			Title: "Reconciling finished",
			Lines: []string{"SlipStream matched the restored library with the torrents and files on your Nodes. Searches and removals run again."},
			Tone:  entry.Good,
		}
	},
}

func Entries() []entry.Entry { return []entry.Entry{RestoredRow, ReplacedRow, FinishedRow} }

// announce records Backup restored and, on a new machine, Controller
// replaced, once per restore.
func (s *Service) announce(ctx context.Context) error {
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		state, err := q.GetState(ctx)
		if errors.Is(err, storage.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if n, err := q.SetAnnounced(ctx, storage.Now()); err != nil || n == 0 {
			return err
		}
		if err := RestoredRow.Record(ctx, tx, *s.cfg.History, RestoredFacts{Backup: state.BackupName, TakenAt: state.BackupTakenAt.Time()}); err != nil {
			return err
		}
		if state.NewMachine == 0 {
			return nil
		}
		named, err := nodes.Names(ctx, tx)
		if err != nil || len(named) == 0 {
			return err
		}
		return ReplacedRow.Record(ctx, tx, *s.cfg.History, ReplacedFacts{Controller: named[0].Name})
	})
	if err != nil {
		return fmt.Errorf("record the restore in History: %w", err)
	}
	return nil
}
