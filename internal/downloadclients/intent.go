package downloadclients

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// Intent is node's Downloading intent: its live Torrents and its
// discards, the Removing ones, each in added order and naming its client,
// its settings with defaults, the thresholds, the disk guard's hold, how
// often to sample the downloads volume, and its external clients, on or
// off, with every setting but their secrets. The same library gives the
// same bytes.
func Intent(ctx context.Context, r storage.Reader[file], node role.NodeID) (*dlv1.Intent, error) {
	rows, err := db.New(r).ListIntent(ctx, node)
	if err != nil {
		return nil, err
	}
	overrides, err := nodeOverridesOf(ctx, r, node)
	if err != nil {
		return nil, err
	}
	thresholds, err := thresholdOverridesOf(ctx, r)
	if err != nil {
		return nil, err
	}
	guards, err := guardsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	every, err := roots.SampleEveryIn(ctx, r)
	if err != nil {
		return nil, err
	}
	_, guarded := guards[node]
	s, th := overrides.Settings(), thresholds.Thresholds()
	intent := &dlv1.Intent{
		HoldDownloads:      guarded,
		SampleEverySeconds: int64(every / time.Second),
		Settings: &dlv1.Settings{
			DownloadLimit:       s.DownloadLimit,
			UploadLimit:         s.UploadLimit,
			ActiveDownloads:     int32(s.ActiveDownloads), //nolint:gosec // the setting's Max fits
			ActiveChecks:        int32(s.ActiveChecks),    //nolint:gosec // the setting's Max fits
			Connections:         int32(s.Connections),     //nolint:gosec // the setting's Max fits
			Utp:                 s.UTP,
			Dht:                 s.DHT,
			StalledAfterSeconds: int64(s.StalledAfter / time.Second),
		},
		Thresholds: &dlv1.Thresholds{
			FaultStallSeconds:      int64(th.FaultStall / time.Second),
			UnregisteredGapSeconds: int64(th.UnregisteredGap / time.Second),
		},
	}
	for _, row := range rows {
		var client []byte
		if row.ClientID != nil {
			client = row.ClientID.Bytes()
		}
		if row.RemovedAt.IsZero() {
			intent.Torrents = append(intent.Torrents, &dlv1.Torrent{
				Id: row.ID.Bytes(), InfoHash: row.InfoHash, Wanted: wantedOnWire[Wanted(row.Wanted)], Order: row.AddedOrder, ClientId: client,
			})
			continue
		}
		intent.Discards = append(intent.Discards, &dlv1.Discard{Id: row.ID.Bytes(), InfoHash: row.InfoHash, DeleteFiles: row.DeleteFiles == 1, ClientId: client})
	}
	if intent.Clients, err = clientsOnWire(ctx, r, node); err != nil {
		return nil, err
	}
	return intent, nil
}

// clientsOnWire is node's external clients in id order, as the intent
// carries them: never a secret.
func clientsOnWire(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]*dlv1.Client, error) {
	rows, err := db.New(r).ListNodeClients(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]*dlv1.Client, len(rows))
	for i, row := range rows {
		e, err := externalOf(row)
		if err != nil {
			return nil, err
		}
		c := &dlv1.Client{
			Id: e.ID.Bytes(), Kind: kindOnWire[e.Kind], Name: e.Name, Address: e.Address.String(), Username: e.Username,
			Label: e.Label.String(), Folder: e.Folder,
			CheckEverySeconds: int64(e.CheckEvery / time.Second), ImportDelaySeconds: int64(e.ImportDelay / time.Second),
			On: e.On, SettingsVersion: e.Version,
		}
		for _, f := range e.Folders.Pairs() {
			c.ClientFolders = append(c.ClientFolders, &dlv1.ClientFolder{Client: f.Client, Node: f.Node})
		}
		if d := e.Daemon; d != nil {
			c.Daemon = &dlv1.Daemon{Host: d.Host, Port: int32(d.Port)} //nolint:gosec // the schema CHECKs the port
		}
		out[i] = c
	}
	return out, nil
}

var wantedOnWire = map[Wanted]dlv1.Wanted{Started: dlv1.Wanted_WANTED_STARTED, Paused: dlv1.Wanted_WANTED_PAUSED}

// Held is a live Torrent as a Node's client keeps it, with its progress
// as the Node last reported it.
type Held struct {
	ID       ID
	InfoHash bencode.InfoHash
	// Name is the metainfo's name.
	Name string
	Done int64
	Size int64
}

// Kept is node's live Torrents in added order.
func Kept(ctx context.Context, r storage.Reader[file], node role.NodeID) ([]Held, error) {
	rows, err := db.New(r).ListKept(ctx, node)
	if err != nil {
		return nil, err
	}
	out := make([]Held, len(rows))
	for i, row := range rows {
		rep, err := reportOf(row.Report)
		if err != nil {
			return nil, fmt.Errorf("torrent %s: %w", row.ID, err)
		}
		out[i] = Held{ID: row.ID, InfoHash: bencode.InfoHash(row.InfoHash), Name: row.Name, Done: rep.done(row.Size), Size: row.Size}
	}
	return out, nil
}

// done is the bytes done the report says, all of size once complete.
func (r *Report) done(size int64) int64 {
	switch {
	case r == nil:
		return 0
	case r.Complete:
		return size
	}
	return r.Done
}

func reportOf(stored *string) (*Report, error) {
	if stored == nil {
		return nil, nil
	}
	var rep Report
	if err := json.Unmarshal([]byte(*stored), &rep); err != nil {
		return nil, fmt.Errorf("report: %w", err)
	}
	return &rep, nil
}

// DownloadingOwner counts the Torrents a Node holds until it parts from
// them: a Removing one too, so a drain lasts until the Node confirms each
// removal. Live Torrents seed until seeding goals exist, so a Node never
// finishes them on its own.
var DownloadingOwner = nodes.Owner{Role: role.Downloading, Held: held}

func held(ctx context.Context, r storage.Reader[file], node role.NodeID) (nodes.Held, error) {
	row, err := db.New(r).CountHeld(ctx, node)
	return nodes.Held{Units: row.Torrents, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: row.Bytes, Endless: row.Live > 0}, err
}
