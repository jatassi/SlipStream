package node

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
)

// healthCategories is every Health category after Storage, one line each,
// in the order health.IDs lists them.
func healthCategories(h *half) []health.Category {
	return []health.Category{
		{About: imports.HealthAbout, Source: h.imports.HealthSource()},
		{About: nodes.HealthAbout, Source: h.nodes.HealthSource()},
		{About: downloadclients.HealthAbout, Source: h.clients.HealthSource()},
		{About: mediaservers.HealthAbout, Source: h.mediaServersHealth},
		{About: reachability.HealthAbout, Source: health.Join(h.reachability.HealthSource(), h.tasks.HealthSource(health.RemoteAccessID))},
		{About: trackers.HealthAbout, Source: health.Join(h.trackers.HealthSource(), h.tasks.HealthSource(health.SeedingID))},
		{About: indexers.HealthAbout, Source: health.Join(h.indexers.HealthSource(), h.tasks.HealthSource(health.IndexersID))},
		{About: subtitles.HealthAbout, Source: health.Join(h.subtitles.HealthSource(), h.tasks.HealthSource(health.SubtitlesID))},
		{About: tasks.HealthAbout, Source: h.tasks.HealthSource(health.ScheduledTasksID)},
		{About: backups.HealthAbout, Source: h.backups.HealthSource()},
		{About: notifications.HealthAbout, Source: h.notifications.HealthSource()},
	}
}

// storageParts is what the Storage category judges: a line per holder
// that samples volumes, and the library roots' own items.
func storageParts(lib *storage.DB[storage.Library], h *half) health.StorageParts {
	scratch := scratchVolumes{
		scratch: h.scratch, roster: h.nodes,
		names: func(ctx context.Context) (names map[role.NodeID]string, err error) {
			err = lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
				names, err = liveNames(ctx, r)
				return err
			})
			return names, err
		},
		reserved: func(ctx context.Context, node role.NodeID, volume string) (n int64, err error) {
			err = lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
				n, err = h.transfers.ReservedIn(ctx, r, node, volume)
				return err
			})
			return n, err
		},
		events: []live.Event{nodes.ChangedEvent, transfers.ChangedEvent},
	}
	return health.StorageParts{
		Volumes:    []health.VolumeSource{h.roots.HealthVolumes(), h.clients.HealthVolumes(), scratch.source()},
		Roots:      h.roots.HealthSource(),
		RootEvents: roots.HealthEvents,
	}
}

func (n *Node) openHealth(ctx context.Context, lib *storage.DB[storage.Library], h *half) (*health.Health, error) {
	return health.Open(ctx, health.Config{
		Library:     lib,
		Hub:         n.hub,
		Changed:     health.ChangedEvent.Bind(n.hub),
		Storage:     storageParts(lib, h),
		Categories:  healthCategories(h),
		Names:       liveNames,
		Roster:      h.nodes,
		Zone:        n.zone(),
		Timings:     n.cfg.Timings.Health,
		Log:         n.rootLog.With("subsystem", "health"),
		Transitions: n.history.Transitions,
	})
}

func liveNames(ctx context.Context, rd storage.Reader[storage.Library]) (map[role.NodeID]string, error) {
	named, err := nodes.Names(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]string, len(named))
	for _, n := range named {
		out[n.ID] = n.Name
	}
	return out, nil
}

// scratchVolumes is the third Storage line: the scratch volume of each
// Node serving Converting, as its holder last reported it, with what
// placed Conversions still reserve there taken off its free space.
type scratchVolumes struct {
	// scratch is node's scratch volume from its facts, capped, nil while
	// unknown.
	scratch func(node role.NodeID) *cv1.Volume
	roster  interface {
		Serving(r role.Name) []role.NodeID
		InStep(node role.NodeID) bool
	}
	names func(ctx context.Context) (map[role.NodeID]string, error)
	// reserved is the bytes placed Conversions on node still expect to
	// write on volume.
	reserved func(ctx context.Context, node role.NodeID, volume string) (int64, error)
	events   []live.Event
}

func (s scratchVolumes) source() health.VolumeSource {
	return health.VolumeSource{Volumes: s.volumes, Sample: s.sample, Events: s.events, Remedy: "move the scratch folder to another volume"}
}

func (s scratchVolumes) volumes(ctx context.Context) ([]health.Volume, error) {
	names, err := s.names(ctx)
	if err != nil {
		return nil, err
	}
	var out []health.Volume
	for _, node := range s.roster.Serving(role.Converting) {
		v := s.scratch(node)
		if v == nil {
			continue
		}
		reserved, err := s.reserved(ctx, node, v.GetId())
		if err != nil {
			return nil, fmt.Errorf("read what Conversions reserve on %s: %w", node, err)
		}
		out = append(out, health.Volume{
			Node: node, NodeName: names[node], Mount: v.GetMount(), Name: v.GetName(),
			Total: v.GetTotal(), Free: max(v.GetFree()-reserved, 0),
			SampledAt: time.UnixMilli(v.GetSampledAtUnixMs()), Live: s.roster.InStep(node),
		})
	}
	return out, nil
}

// sample is nothing for a Node in step, whose holder reports its scratch
// volume every progress tick.
func (s scratchVolumes) sample(_ context.Context, node role.NodeID) error {
	if !s.roster.InStep(node) {
		return fmt.Errorf("%w: %s is away", health.ErrUnreachable, node)
	}
	return nil
}
