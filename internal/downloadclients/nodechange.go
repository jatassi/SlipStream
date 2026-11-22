package downloadclients

import (
	"context"
	"fmt"
	"path"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

const firstMissing = 20

// MissingTorrent is a tracked torrent whose files the client's new Node
// didn't find at their size.
type MissingTorrent struct {
	ID   ID
	Name string
}

// FilesMissingError refuses a client's Node change while the new Node
// doesn't find some of its torrents' files: how many, and the first of
// them.
type FilesMissingError struct {
	Node  role.NodeID
	First []MissingTorrent
	Count int
}

func (e *FilesMissingError) Error() string {
	return fmt.Sprintf("the new Node didn't find the files of %d torrents", e.Count)
}

// MovesUseError refuses a client's Node change while Count Moves use it.
type MovesUseError struct {
	Name  string
	Count int64
}

func (e *MovesUseError) Error() string {
	return fmt.Sprintf("%d Moves use %s", e.Count, e.Name)
}

// HeldOnNodeError refuses a client's Node change while the new Node
// already holds some of its torrents live: their names.
type HeldOnNodeError struct {
	Node  role.NodeID
	Names []string
}

func (e *HeldOnNodeError) Error() string {
	return fmt.Sprintf("node %s already holds %d of the client's torrents", e.Node, len(e.Names))
}

// notMoving is *MovesUseError while Moves use client e.
func (c *Clients) notMoving(ctx context.Context, r storage.Reader[file], e External) error {
	n, err := c.cfg.Moving.UsesClient(ctx, r, e.ID)
	if err != nil || n == 0 {
		return err
	}
	return &MovesUseError{Name: e.Name, Count: n}
}

// look has client now's Node find each live torrent's files, by size,
// through now's Client folders: under base, the client's download folder
// on that Node, in its Content Type's folder, or at the save folder it
// reported. Any it doesn't find is *FilesMissingError.
func (c *Clients) look(ctx context.Context, now External, base string) error {
	var rows []db.ListLookableRow
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		rows, err = db.New(r).ListLookable(ctx, &now.ID)
		return err
	})
	if err != nil {
		return err
	}
	missing := &FilesMissingError{Node: now.Node}
	for _, row := range rows {
		found, err := c.lookOne(ctx, now, base, row)
		if err != nil {
			return err
		}
		if found {
			continue
		}
		if missing.Count < firstMissing {
			missing.First = append(missing.First, MissingTorrent{ID: row.ID, Name: row.Name})
		}
		missing.Count++
	}
	if missing.Count > 0 {
		return missing
	}
	return nil
}

func (c *Clients) lookOne(ctx context.Context, now External, base string, row db.ListLookableRow) (bool, error) {
	m, err := bencode.DecodeMetainfo(row.Metainfo)
	if err != nil {
		return false, fmt.Errorf("read the files of Torrent %s: %w", row.ID, err)
	}
	folder, under := base, ""
	if row.ContentType != nil {
		desc, _ := c.cfg.Types.Describe(*row.ContentType)
		under = desc.Name
	}
	if row.SaveFolder != nil {
		onNode, _, ok := now.Folders.OnNode(*row.SaveFolder, '/')
		if !ok {
			return false, nil
		}
		folder, under = onNode, ""
	}
	req := &dlv1.LookRequest{Folder: folder}
	for _, f := range m.Info.Files {
		req.Entries = append(req.Entries, &dlv1.LookEntry{Name: path.Join(append([]string{under, m.Info.Name}, f.Path...)...), Size: f.Length})
	}
	res, err := c.cfg.Remote(now.Node).Look(ctx, req)
	if err != nil {
		return false, &UnavailableError{Node: now.Node, Err: err}
	}
	for _, found := range res.GetFound() {
		if !found {
			return false, nil
		}
	}
	return len(res.GetFound()) == len(req.Entries), nil
}

// rerecord moves client now's Torrents to its new Node in the Write that
// saves it there, unless that Node holds some of them live already.
func rerecord(ctx context.Context, tx *storage.Tx[file], now External) error {
	q := db.New(tx)
	held, err := q.ListHeldElsewhere(ctx, db.ListHeldElsewhereParams{ClientID: &now.ID, NodeID: now.Node})
	if err != nil {
		return err
	}
	if len(held) > 0 {
		return &HeldOnNodeError{Node: now.Node, Names: held}
	}
	return q.RerecordClient(ctx, db.RerecordClientParams{NodeID: now.Node, ClientID: &now.ID})
}
