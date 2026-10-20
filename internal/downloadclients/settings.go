package downloadclients

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// libtorrent keeps a rate limit in an int.
const maxRate = 2000 * 1024 * 1024

var (
	DownloadLimit   = setting.Rate{Key: "downloading.downloadLimit", Label: "Download limit", Max: maxRate, Zero: "No limit"}
	UploadLimit     = setting.Rate{Key: "downloading.uploadLimit", Label: "Upload limit", Max: maxRate, Zero: "No limit"}
	ActiveDownloads = setting.Count{Key: "downloading.activeDownloads", Label: "Downloads at once", Default: 3, Min: 0, Max: 1000, Unit: setting.Noun{One: "download", Many: "downloads"}, Zero: "No limit"}
	ActiveChecks    = setting.Count{Key: "downloading.activeChecks", Label: "File checks at once", Default: 1, Min: 1, Max: 50, Unit: setting.Noun{One: "check", Many: "checks"}}
	Connections     = setting.Count{Key: "downloading.connections", Label: "Peer connections", Default: 8000, Min: 50, Max: 65535, Unit: setting.Noun{One: "connection", Many: "connections"}}
	UTP             = setting.Switch{Key: "downloading.utp", Label: "uTP, incoming and outgoing", Default: true}
	StalledAfter    = setting.Duration{Key: "downloading.stalledAfter", Label: "Stalled after", Default: 10 * time.Minute, Min: time.Minute, Max: 24 * time.Hour, Step: time.Minute}
	FaultStall      = setting.Duration{Key: "downloading.faultStall", Label: "A download with no data at all is bad after", Default: 12 * time.Hour, Min: time.Hour, Max: 30 * 24 * time.Hour, Step: time.Hour}
	UnregisteredGap = setting.Duration{Key: "downloading.unregisteredGap", Label: "Unregistered twice, this far apart", Default: 30 * time.Minute, Min: time.Minute, Max: 7 * 24 * time.Hour, Step: time.Minute}
)

// NodeSettings is one Node's client settings with every default applied.
type NodeSettings struct {
	// DownloadLimit and UploadLimit are bytes a second, 0 for none.
	DownloadLimit, UploadLimit int64
	// ActiveDownloads is 0 for none.
	ActiveDownloads           int64
	ActiveChecks, Connections int64
	UTP                       bool
	StalledAfter              time.Duration
}

// NodeOverrides is one Node's stored overrides, each nil at its default.
type NodeOverrides struct {
	DownloadLimit, UploadLimit, ActiveDownloads, ActiveChecks, Connections, StalledAfter *int64
	UTP                                                                                  *bool
}

// NodeChange is the per-Node settings to store, each nil to leave.
type NodeChange struct {
	DownloadLimit, UploadLimit, ActiveDownloads, ActiveChecks, Connections, StalledAfter *setting.Stored[int64]
	UTP                                                                                  *setting.Stored[bool]
}

func (o NodeOverrides) Settings() NodeSettings {
	return NodeSettings{
		DownloadLimit:   DownloadLimit.Value(o.DownloadLimit),
		UploadLimit:     UploadLimit.Value(o.UploadLimit),
		ActiveDownloads: ActiveDownloads.Value(o.ActiveDownloads),
		ActiveChecks:    ActiveChecks.Value(o.ActiveChecks),
		Connections:     Connections.Value(o.Connections),
		UTP:             UTP.Value(o.UTP),
		StalledAfter:    StalledAfter.Value(o.StalledAfter),
	}
}

// nodeOverridesOf reads node's overrides. A Node with no row has none.
func nodeOverridesOf(ctx context.Context, r storage.Reader[file], node role.NodeID) (NodeOverrides, error) {
	row, err := db.New(r).GetNodeSettings(ctx, node)
	if errors.Is(err, storage.ErrNoRows) {
		return NodeOverrides{}, nil
	}
	if err != nil {
		return NodeOverrides{}, err
	}
	return NodeOverrides{
		DownloadLimit: row.DownloadLimit, UploadLimit: row.UploadLimit, ActiveDownloads: row.ActiveDownloads,
		ActiveChecks: row.ActiveChecks, Connections: row.Connections, StalledAfter: row.StalledAfterSeconds, UTP: row.Utp,
	}, nil
}

// NodeSettings is node's stored overrides.
func (c *Clients) NodeSettings(ctx context.Context, node role.NodeID) (NodeOverrides, error) {
	return nodeOverridesOf(ctx, c.cfg.Library, node)
}

// SetNodeSettings stores ch as node's settings, which node applies from
// its next intent.
func (c *Clients) SetNodeSettings(ctx context.Context, node role.NodeID, ch NodeChange) (NodeOverrides, error) {
	var o NodeOverrides
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		o, err = setNodeSettings(ctx, tx, node, ch)
		return err
	})
	if err != nil {
		return NodeOverrides{}, err
	}
	c.cfg.Touch(node)
	return o, nil
}

func setNodeSettings(ctx context.Context, tx *storage.Tx[file], node role.NodeID, c NodeChange) (NodeOverrides, error) {
	o, err := nodeOverridesOf(ctx, tx, node)
	if err != nil {
		return NodeOverrides{}, err
	}
	for _, f := range []struct {
		change *setting.Stored[int64]
		column **int64
	}{
		{c.DownloadLimit, &o.DownloadLimit},
		{c.UploadLimit, &o.UploadLimit},
		{c.ActiveDownloads, &o.ActiveDownloads},
		{c.ActiveChecks, &o.ActiveChecks},
		{c.Connections, &o.Connections},
		{c.StalledAfter, &o.StalledAfter},
	} {
		if f.change != nil {
			*f.column = f.change.Override()
		}
	}
	if c.UTP != nil {
		o.UTP = c.UTP.Override()
	}
	return o, db.New(tx).SetNodeSettings(ctx, db.SetNodeSettingsParams{
		NodeID: node, DownloadLimit: o.DownloadLimit, UploadLimit: o.UploadLimit, ActiveDownloads: o.ActiveDownloads,
		ActiveChecks: o.ActiveChecks, Connections: o.Connections, Utp: o.UTP, StalledAfterSeconds: o.StalledAfter,
	})
}

// Thresholds decide when a release counts as bad, on every Node.
type Thresholds struct{ FaultStall, UnregisteredGap time.Duration }

// ThresholdOverrides is the stored thresholds, each nil at its default.
type ThresholdOverrides struct{ FaultStall, UnregisteredGap *int64 }

// ThresholdChange is the thresholds to store, each nil to leave.
type ThresholdChange struct{ FaultStall, UnregisteredGap *setting.Stored[int64] }

func (o ThresholdOverrides) Thresholds() Thresholds {
	return Thresholds{FaultStall: FaultStall.Value(o.FaultStall), UnregisteredGap: UnregisteredGap.Value(o.UnregisteredGap)}
}

func ThresholdOverridesOf(ctx context.Context, r storage.Reader[file]) (ThresholdOverrides, error) {
	row, err := db.New(r).GetThresholds(ctx)
	return ThresholdOverrides{FaultStall: row.FaultStallSeconds, UnregisteredGap: row.UnregisteredGapSeconds}, err
}

func SetThresholds(ctx context.Context, tx *storage.Tx[file], c ThresholdChange) error {
	o, err := ThresholdOverridesOf(ctx, tx)
	if err != nil {
		return err
	}
	if c.FaultStall != nil {
		o.FaultStall = c.FaultStall.Override()
	}
	if c.UnregisteredGap != nil {
		o.UnregisteredGap = c.UnregisteredGap.Override()
	}
	return db.New(tx).SetThresholds(ctx, db.SetThresholdsParams{FaultStallSeconds: o.FaultStall, UnregisteredGapSeconds: o.UnregisteredGap})
}
