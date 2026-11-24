package reachability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// The Remote access section: one stored value each. The cadence is the
// task's own, CheckEvery.
var (
	WarnAfter = setting.Duration{
		Key: "reachability.warnAfter", Label: "Warn in Health after a route fails for",
		Default: 15 * time.Minute, Min: time.Minute, Max: 7 * 24 * time.Hour, Step: time.Minute,
	}
	WatchEvery = setting.Duration{
		Key: "reachability.watchEvery", Label: "Look for network changes every",
		Default: 5 * time.Minute, Min: time.Minute, Max: 24 * time.Hour, Step: time.Minute,
		Floor: "Operators of public STUN servers ask clients not to send requests more often.",
	}
	ChangeWait = setting.Duration{
		Key: "reachability.changeWait", Label: "Wait after a network change",
		Default: time.Minute, Min: 0, Max: time.Hour, Step: time.Second,
	}
	// STUNServers unset means each Node's own stun entry; empty means
	// contact none.
	STUNServers = setting.List{Key: "reachability.stunServers", Label: "STUN servers", Item: setting.HostPort}
)

// The per-Node settings of a Node's Network section.
var (
	MapPorts   = setting.Switch{Key: "reachability.mapPorts", Label: "Open router ports automatically"}
	StableIPv6 = setting.IP{Key: "reachability.stableIPv6", Label: "Stable IPv6 address", Family: setting.IPv6Only}
)

// settings is the section as one run reads it.
type settings struct {
	warnAfter, watchEvery, changeWait time.Duration
	stun                              *[]string
}

func settingsIn(ctx context.Context, rd storage.Reader[storage.Library]) (settings, db.ReachabilitySetting, error) {
	row, err := db.New(rd).GetSettings(ctx)
	if err != nil {
		return settings{}, row, err
	}
	s := settings{warnAfter: WarnAfter.Value(row.WarnAfterSeconds), watchEvery: WatchEvery.Value(row.WatchEverySeconds), changeWait: ChangeWait.Value(row.ChangeWaitSeconds)}
	if row.StunServers != nil {
		var list []string
		if err := json.Unmarshal([]byte(*row.StunServers), &list); err != nil {
			return s, row, fmt.Errorf("read the STUN servers: %w", err)
		}
		s.stun = &list
	}
	return s, row, nil
}

// Panel is the Remote access section of Settings › Access.
func (r *Reachability) Panel() setting.Panel { return panel{r} }

type panel struct{ r *Reachability }

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	var (
		s   settings
		row db.ReachabilitySetting
	)
	err := p.r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		s, row, err = settingsIn(ctx, rd)
		return err
	})
	if err != nil {
		return nil, err
	}
	list := STUNServers.Field(s.stun)
	list.DefaultFrom = "Each Node's own STUN servers"
	return []setting.Field{WarnAfter.Field(row.WarnAfterSeconds), WatchEvery.Field(row.WatchEverySeconds), ChangeWait.Field(row.ChangeWaitSeconds), list}, nil
}

// write is one stored change a patch makes.
type write func(context.Context, *db.Queries) error

var durations = map[string]struct {
	setting.Duration
	set func(*db.Queries, context.Context, *int64) error
}{
	WarnAfter.Key:  {WarnAfter, (*db.Queries).SetWarnAfter},
	WatchEvery.Key: {WatchEvery, (*db.Queries).SetWatchEvery},
	ChangeWait.Key: {ChangeWait, (*db.Queries).SetChangeWait},
}

// Apply stores the patch, all or none. Every Node's intent follows the
// watch, the wait and the STUN servers.
func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	writes := make([]write, 0, len(patch))
	for key, raw := range patch {
		w, err := sectionWrite(key, raw)
		if err != nil {
			return err
		}
		writes = append(writes, w)
	}
	if err := p.r.writeAll(ctx, writes); err != nil {
		return err
	}
	p.r.touch(ctx)
	p.r.changed(ctx)
	return nil
}

func sectionWrite(key string, raw json.RawMessage) (write, error) {
	if key == STUNServers.Key {
		v, err := STUNServers.ParseJSON(raw)
		if err != nil {
			return nil, err
		}
		var text *string
		if list := v.Override(); list != nil {
			s, err := encode(*list)
			if err != nil {
				return nil, err
			}
			text = &s
		}
		return func(ctx context.Context, q *db.Queries) error { return q.SetSTUNServers(ctx, text) }, nil
	}
	d, ok := durations[key]
	if !ok {
		return nil, &setting.UnknownKeyError{Key: key}
	}
	v, err := d.ParseJSON(raw)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, q *db.Queries) error { return d.set(q, ctx, v.Override()) }, nil
}

func (r *Reachability) writeAll(ctx context.Context, writes []write) error {
	return r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		for _, w := range writes {
			if err := w(ctx, q); err != nil {
				return err
			}
		}
		return nil
	})
}

// NodePanel is a Node's two settings in its Network section.
func (r *Reachability) NodePanel(node role.NodeID) setting.Panel { return nodePanel{r, node} }

type nodePanel struct {
	r    *Reachability
	node role.NodeID
}

func (p nodePanel) read(ctx context.Context) (db.ReachabilityNodeSetting, error) {
	var row db.ReachabilityNodeSetting
	err := p.r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		row, err = db.New(rd).GetNodeSetting(ctx, p.node)
		if errors.Is(err, storage.ErrNoRows) {
			row, err = db.ReachabilityNodeSetting{NodeID: p.node}, nil
		}
		return err
	})
	return row, err
}

func (p nodePanel) Fields(ctx context.Context) ([]setting.Field, error) {
	if err := p.r.live(ctx, p.node); err != nil {
		return nil, err
	}
	row, err := p.read(ctx)
	if err != nil {
		return nil, err
	}
	var mapPorts *bool
	if row.MapPorts != nil {
		mapPorts = new(*row.MapPorts == 1)
	}
	var pin *netip.Addr
	if row.StableIpv6 != nil {
		a, err := netip.ParseAddr(*row.StableIpv6)
		if err != nil {
			return nil, fmt.Errorf("read the stable IPv6 address: %w", err)
		}
		pin = &a
	}
	return []setting.Field{MapPorts.Field(mapPorts), StableIPv6.Field(pin)}, nil
}

func (p nodePanel) Apply(ctx context.Context, patch setting.Patch) error {
	if err := p.r.live(ctx, p.node); err != nil {
		return err
	}
	writes := make([]write, 0, len(patch))
	for key, raw := range patch {
		w, err := p.write(key, raw)
		if err != nil {
			return err
		}
		writes = append(writes, w)
	}
	if err := p.r.writeAll(ctx, writes); err != nil {
		return err
	}
	p.r.network.Touch(p.node)
	p.r.changed(ctx)
	if _, ok := patch[MapPorts.Key]; ok {
		return p.r.recheck(ctx, p.node)
	}
	return nil
}

// recheck runs node's units again with a fresh pass, which maps or
// deletes what its intent now asks.
func (r *Reachability) recheck(ctx context.Context, node role.NodeID) error {
	r.mu.Lock()
	r.freshAfter[node] = r.now()
	r.mu.Unlock()
	units, err := r.unitsOf(ctx, node)
	if err != nil {
		return err
	}
	r.start(ctx, units...)
	return nil
}

func (p nodePanel) write(key string, raw json.RawMessage) (write, error) {
	switch key {
	case MapPorts.Key:
		v, err := MapPorts.ParseJSON(raw)
		if err != nil {
			return nil, err
		}
		var on *int64
		if b := v.Override(); b != nil {
			on = new(int64(btoi(*b)))
		}
		return func(ctx context.Context, q *db.Queries) error {
			return q.SetMapPorts(ctx, db.SetMapPortsParams{NodeID: p.node, MapPorts: on})
		}, nil
	case StableIPv6.Key:
		v, err := StableIPv6.ParseJSON(raw)
		if err != nil {
			return nil, err
		}
		var text *string
		if a := v.Override(); a != nil {
			if !p.r.cfg.Rules.Global(*a) {
				return nil, &setting.RangeError{Key: StableIPv6.Key, Sentence: "Use a global IPv6 address. Addresses starting with fd work only inside your home."}
			}
			text = new(a.String())
		}
		return func(ctx context.Context, q *db.Queries) error {
			return q.SetStableIPv6(ctx, db.SetStableIPv6Params{NodeID: p.node, StableIpv6: text})
		}, nil
	}
	return nil, &setting.UnknownKeyError{Key: key}
}

// live refuses a Node that was never paired or was removed.
func (r *Reachability) live(ctx context.Context, node role.NodeID) error {
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		return err
	}
	if n, ok := named[node]; !ok || n.Removed {
		return ErrNodeNotFound
	}
	return nil
}
