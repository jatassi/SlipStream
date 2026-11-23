package moves

import (
	"context"
	"errors"
	"path"
	"strconv"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// Purpose is moves' Transfers purpose: a unit's files crossing to another
// Node, one group per pass, so a final pass's delta verifies apart from
// the files placed before it.
func (m *Moves) Purpose() transfers.Purpose {
	return transfers.Purpose{Kind: transfers.Move, Verified: m.verified, Failed: m.copyFailed, After: m.copied, Names: transferNames}
}

func groupOf(pass int) string { return "pass-" + strconv.Itoa(pass) }

// copyRequest is the Transfer u wants: each of its files not placed yet,
// in its pass's group, at Low priority. None ends the Transfer.
func copyRequest(u Unit) transfers.Request {
	r := transfers.Request{
		Purpose: transfers.Move, Job: u.ID.String(), Source: u.Subject.From.Node,
		Destination: transfers.Root{ID: u.Subject.To.Root}, Priority: transfers.Low,
	}
	for _, f := range u.Files {
		if f.Placed == nil && f.Via != ViaArriving {
			r.Files = append(r.Files, transfers.FileRequest{Group: groupOf(u.pass()), Name: path.Base(f.Sits.String()), Source: sourceOf(u, f), Size: f.Size})
		}
	}
	return r
}

// copy sends u's unplaced files to its destination Node in tx: it copies
// until they verify, or places at once when every file is placed. The
// caller Touches transfers once tx commits.
func (m *Moves) copy(ctx context.Context, tx *storage.Tx[file], u *Unit) error {
	pass := u.pass()
	id, err := m.cfg.Transfers.Want(ctx, tx, copyRequest(*u))
	if err != nil {
		return err
	}
	if id.IsZero() {
		u.Stage, u.Due = Placing{Pass: pass}, new(m.now())
		return nil
	}
	u.Stage, u.Due = Copying{Transfer: id, Pass: pass}, nil
	return nil
}

// endCopy ends unit id's Transfer in tx, so its destination drops what it
// staged.
func (m *Moves) endCopy(ctx context.Context, tx *storage.Tx[file], id UnitID) error {
	_, err := m.cfg.Transfers.Want(ctx, tx, transfers.Request{Purpose: transfers.Move, Job: id.String()})
	return err
}

// stagedCopy is a file's verified copy that its unit's Transfer staged
// in the destination root, at its position there.
type stagedCopy struct {
	transfer transfers.ID
	position int
}

// staged is where file f of u waits in its destination root, nil when
// u's Transfer holds no verified copy of it.
func (m *Moves) staged(ctx context.Context, u Unit, f File) (*stagedCopy, error) {
	var t transfers.Transfer
	var found bool
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		t, found, err = m.cfg.Transfers.Find(ctx, r, transfers.Move, u.ID.String())
		return err
	})
	if err != nil || !found {
		return nil, err
	}
	ref, group := sourceOf(u, f), groupOf(u.pass())
	for _, tf := range t.Files {
		if tf.Group == group && tf.Source == ref && tf.Verified && !tf.Placed {
			return &stagedCopy{transfer: t.ID, position: tf.Position}, nil
		}
	}
	return nil, nil
}

// verified is the Verified hook: the pass's files are staged whole at the
// destination, and the unit places them.
func (m *Moves) verified(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, group string) error {
	return m.hooked(ctx, tx, t, func(u *Unit, s Copying) error {
		if group == groupOf(s.Pass) {
			u.Stage, u.Due = Placing{Pass: s.Pass, Transfer: &t.ID}, new(m.now())
		}
		return nil
	})
}

// copyFailed is the Failed hook: the unit fails for the Transfer's
// reason, and its undo ends the Transfer.
func (m *Moves) copyFailed(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, f transfers.Failure) error {
	return m.hooked(ctx, tx, t, func(u *Unit, _ Copying) error {
		return m.fail(ctx, tx, u, failureOf(f.Code), f.Sentence)
	})
}

func failureOf(c transfers.Code) FailureCode {
	switch c {
	case transfers.SourceGone:
		return SourceGone
	case transfers.SourceFailed:
		return SourceFailed
	case transfers.Damaged:
		return Damaged
	case transfers.SourceWrittenOff:
		return SourceWrittenOff
	case transfers.Rerouted, transfers.FolderFailed:
	}
	return DestinationGone
}

// hooked runs change over the unit t copies for, in a Transfers hook's
// Write, while it copies by t. Any other unit, or none, is nothing to do.
func (m *Moves) hooked(ctx context.Context, tx *storage.Tx[file], t transfers.Transfer, change func(u *Unit, s Copying) error) error {
	id, err := storage.ParseUUID(t.Job)
	if err != nil {
		return err
	}
	u, err := load(ctx, tx, UnitID{UUID: id})
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	s, ok := u.Stage.(Copying)
	if !ok || s.Transfer != t.ID {
		return nil
	}
	if err := change(&u, s); err != nil {
		return err
	}
	if err := save(ctx, tx, u); err != nil {
		return err
	}
	m.mu.Lock()
	m.copiedUnits = append(m.copiedUnits, u)
	m.mu.Unlock()
	return nil
}

// copied is the After hook: it publishes the units the hooks changed and
// has the runner pass.
func (m *Moves) copied([]string) {
	m.mu.Lock()
	us := m.copiedUnits
	m.copiedUnits = nil
	m.mu.Unlock()
	if len(us) > 0 {
		m.cfg.Changed.Publish(context.Background(), changedOf(us...))
	}
	m.Poke()
}

// transferNames is the Names hook: each job's unit named by its Work's
// label, read in r.
func transferNames(ctx context.Context, r storage.Reader[file], jobs []string) (map[string]string, error) {
	out := make(map[string]string, len(jobs))
	for _, job := range jobs {
		id, err := storage.ParseUUID(job)
		if err != nil {
			return nil, err
		}
		u, err := load(ctx, r, UnitID{UUID: id})
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		w, err := works.SummaryOf(ctx, r, u.Subject.Work)
		if err != nil {
			return nil, err
		}
		out[job] = w.Label
	}
	return out, nil
}
