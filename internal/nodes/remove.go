package nodes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// MaxNameLength is the most characters a Node's name has.
const MaxNameLength = 63

var (
	ErrNameTaken   = errors.New("another Node has this name")
	ErrNameInvalid = errors.New("a Node's name is 1 to 63 characters, with no space at either end")
)

func validName(name string) bool {
	return name != "" && utf8.ValidString(name) && utf8.RuneCountInString(name) <= MaxNameLength && strings.TrimSpace(name) == name
}

// Rename names node name, which no other Node that isn't removed has. The
// Node's intent carries the name, so Rename answers once a Node in step
// has it.
func (s *Service) Rename(ctx context.Context, id ID, name string) (Node, error) {
	if !validName(name) {
		return Node{}, ErrNameInvalid
	}
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if _, err := q.GetNode(ctx, id); errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		taken, err := q.NameTaken(ctx, db.NameTakenParams{Name: name, ID: id})
		if err != nil {
			return err
		}
		if taken {
			return ErrNameTaken
		}
		return q.RenameNode(ctx, db.RenameNodeParams{Name: name, ID: id})
	})
	if err != nil {
		return Node{}, fmt.Errorf("rename node %s: %w", id, err)
	}
	if err := s.reload(ctx); err != nil {
		return Node{}, err
	}
	s.controls.Touch(id)
	if err := s.AwaitDelivery(ctx, id); err != nil {
		return Node{}, err
	}
	return s.Get(ctx, id)
}

// entomb turns node's row into a tombstone inside Apply's Write: its dial
// rows go, its key moves to removed_key and its name is free.
func entomb(ctx context.Context, q *db.Queries, node ID) error {
	if err := q.DeleteDials(ctx, node); err != nil {
		return err
	}
	return q.RemoveNode(ctx, db.RemoveNodeParams{RemovedAt: storage.Now(), ID: node})
}

// forget stops pinning a removed node's key, so its Hello is refused as
// removed, and closes its session.
func (s *Service) forget(ctx context.Context, node ID) {
	s.mu.Lock()
	key, ok := s.keys[node]
	if ok {
		delete(s.keys, node)
		delete(s.byKey, key)
		s.removed[key] = true
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	s.reconcileDials(ctx)
	s.cfg.Link.Drop(key)
}

// leaves is the Node o removes, if any.
func (o Outcome) leaves() (ID, bool) {
	for _, e := range o.Effects {
		if l, ok := e.(Leaves); ok {
			return l.Node.ID, true
		}
	}
	return ID{}, false
}
