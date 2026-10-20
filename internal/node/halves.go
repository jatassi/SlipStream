package node

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1/housekeepingv1connect"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/serving"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// peers is how the link treats a peer under the Node's claim: whom it
// admits, and how it answers a Pair.
type peers interface {
	link.Trust
	Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error)
}

// half is the claimed part of a running Node. Each Node starts at most one,
// from Unclaimed or at start, and never restarts it in process.
type half struct {
	peers    peers
	dispatch *rpc.Dispatch
	holding  *role.Holding
	// systems reaches every Node's system, on a Controller.
	systems  *nodesystem.Nodes
	nodes    *nodes.Service
	controls *role.Controls
	cluster  *rpc.Cluster
	works    *works.Works
	metadata *metadata.Metadata
	indexers *indexers.Indexers
	targets  *targets.Targets
	roots    *roots.Roots
	files    *files.Files
	grabs    *grabs.Grabs
	tasks    *tasks.Scheduler
	// transfers receives files on the Controller, and source serves them
	// on a Worker.
	transfers *transfers.Receiver
	source    *transfers.Source
	// serving proxies to the Controller while a Worker holds Serving.
	serving *serving.Worker
}

// startController starts the Controller half: the control half, its own
// holder half in process, and call dispatch over the roster.
func (n *Node) startController(ctx context.Context) error {
	lib, err := n.openLibrary(ctx)
	if err != nil {
		return err
	}
	client := n.httpClient()
	holder := n.openHolder(client)
	opened := n.openLibraries()
	downloads := downloading.Def.Serve(grabs.DownloadIntent, grabs.ApplyDownload)
	stored := storing.Def.Serve(roots.StorageIntent(files.RootContents{}), roots.ApplyStorage)
	arrivals := roots.NewArrivals()
	var (
		systems *nodesystem.Nodes
		library *roots.Roots
	)
	side, err := nodes.Open(ctx, nodes.Config{
		Library: lib.db,
		Key:     n.key.Public(),
		Name:    n.cfg.Name,
		Node:    n.info,
		Link:    n.link,
		Changed: nodes.ChangedEvent.Bind(n.hub),
		Traffic: nodes.TrafficEvent.Bind(n.hub),

		SettingsChanged: nodes.SettingsChangedEvent.Bind(n.hub),
		Served:          append([]role.Served{downloads, stored}, n.env.Test.Served...),
		Owners:          owners(),
		Downloads:       downloads,
		Kept:            grabs.Kept,
		Storage: func(ctx context.Context, r storage.Reader[storage.Library], id role.NodeID) (nodes.StorageFigures, error) {
			return library.Figures(ctx, r, id)
		},
		Facts: arrivals.Note,
		Reach: func(ctx context.Context, id role.NodeID, t nodesystem.Target) (nodesystem.Reached, error) {
			return systems.Reach(ctx, id, t)
		},
		Always:  n.env.Test.Always,
		Timings: n.cfg.Timings.Nodes,
		Stall:   n.cfg.Timings.Role.Stall,
		Log:     n.rootLog.With("subsystem", "nodes"),
	})
	if err != nil {
		return err
	}
	controls := side.Controls()
	holding, local, err := n.startHolding(ctx, role.InProcess(controls), nil)
	if err != nil {
		return err
	}
	self := n.key.Public()
	dispatch := rpc.NewDispatch(n.link, self, self, holding.Holds)
	cluster := rpc.NewCluster(dispatch, side)
	searchv1connect.SearchingService.Mount(dispatch, indexers.NewHome(client, n.cfg.Timings.Indexers))
	metadatav1connect.MetadataService.Mount(dispatch, metadata.NewHolder(holder))
	mediav1connect.MediaService.Mount(dispatch, media.NewService(n.env.Media, n.downloads, n.cfg.Timings.Media))
	notifyv1connect.NotifyingService.Mount(dispatch, notifying.Serve(n.cfg.Timings.Mail, n.env.Net.DialContext))
	storagev1connect.StorageService.Mount(dispatch, n.store)
	n.serveSystem(dispatch)
	providers := make([]metadata.Provider, len(holder.Testers))
	for i, t := range holder.Testers {
		providers[i] = t.Provider()
	}
	typeChanged := works.TypeChangedEvent.Bind(n.hub)
	meta, err := metadata.Open(ctx, metadata.Config{
		Library: lib.db,
		Sealer:  lib.sealer,
		Roster:  side,
		Service: func(node role.NodeID) metadatav1connect.MetadataServiceHandler {
			return metadatav1connect.MetadataService.On(cluster, node)
		},
		Providers:   providers,
		Changed:     metadata.ChangedEvent.Bind(n.hub),
		Timings:     n.cfg.Timings.Metadata,
		Rand:        n.cacheRand,
		Types:       n.env.Types.IDs(),
		TypeChanged: typeChanged,
		Minter:      lib.minter,
		Local:       local,
		ArtworkDir:  n.env.DataDir.ArtworkDir(),
	})
	if err != nil {
		return err
	}
	tree, err := works.Open(ctx, works.Config{
		Library:      lib.db,
		Catalogs:     contenttype.Project(opened, func(t contenttype.Library) works.Catalog { return t }),
		Source:       meta,
		Artwork:      meta,
		Timings:      n.cfg.Timings.Works,
		Changed:      works.ChangedEvent.Bind(n.hub),
		TypesChanged: works.ContentTypesChangedEvent.Bind(n.hub),
		TypeChanged:  typeChanged,
		Log:          n.rootLog.With("subsystem", "works"),
	})
	if err != nil {
		return err
	}
	systems = nodesystem.NewNodes(
		func(id role.NodeID) systemv1connect.NodeSystemServiceHandler {
			return systemv1connect.NodeSystemService.On(cluster, id)
		},
		func(ctx context.Context, id role.NodeID) (string, error) {
			node, err := side.Get(ctx, id)
			if errors.Is(err, nodes.ErrNotFound) {
				return "", nodesystem.ErrNotFound
			}
			return node.Name, err
		},
		nodesystem.FollowTimings{RetryMin: n.cfg.Timings.Link.RedialMin, RetryMax: n.cfg.Timings.Link.RedialMax},
		n.systemRand,
	)
	housekeepingv1connect.HousekeepingService.Mount(dispatch, housekeeping.Service{Local: local})
	lib.mail.SetCourier(notifying.Courier{Roster: side, Self: side.Self(), Cluster: cluster})
	n.lockstep.Handle(controls.Handler())
	n.lockstep.Handle("/", dispatch)
	types := n.env.Types.IDs()
	registry := indexers.New(indexers.Config{
		Library: lib.db,
		Sealer:  lib.sealer,
		Roster:  side,
		Searching: func(node role.NodeID) searchv1connect.SearchingServiceHandler {
			return searchv1connect.SearchingService.On(cluster, node)
		},
		Changed: indexers.ChangedEvent.Bind(n.hub),
	})
	goals, err := targets.Open(ctx, targets.Config{
		Library: lib.db,
		Works:   tree,
		Types:   contenttype.Project(opened, func(t contenttype.Library) targets.Facet { return t }),
		Changed: targets.ChangedEvent.Bind(n.hub),
	})
	if err != nil {
		return err
	}
	library, err = roots.Open(ctx, roots.Config{
		Library: lib.db,
		Types:   types,
		Roster:  side,
		Storage: stored,
		Remote: func(id role.NodeID) storagev1connect.StorageServiceHandler {
			return storagev1connect.StorageService.On(cluster, id)
		},
		Arrivals: arrivals,
		Contents: files.RootContents{},
		Inflight: grabs.RootGrabs{},
		Changed:  roots.ChangedEvent.Bind(n.hub),
		Volumes:  roots.VolumesChangedEvent.Bind(n.hub),
		Log:      n.rootLog.With("subsystem", "roots"),
	})
	if err != nil {
		return err
	}
	shelf := files.New(files.Config{
		Library: lib.db, Works: tree, Targets: goals, Grabs: grabs.GrabCells{},
		Naming:  contenttype.Project(opened, func(t contenttype.Library) naming.Facet { return t }),
		Today:   n.today,
		Disk:    library,
		Timings: n.cfg.Timings.Files,
		Rand:    n.filesRand,
		Log:     n.rootLog.With("subsystem", "files"),
	})
	receiver := transfers.NewReceiver(transfers.Config{Local: local, Link: n.link, Keys: side, Timings: n.cfg.Timings.Transfers})
	pipeline := grabs.New(grabs.Config{
		Library:   lib.db,
		Works:     tree,
		Targets:   goals,
		Files:     shelf,
		Indexers:  registry,
		Terms:     contenttype.Project(opened, func(t contenttype.Library) indexers.Facet { return t }),
		Roster:    side,
		Downloads: downloads,
		Media: func(node role.NodeID) mediav1connect.MediaServiceHandler {
			return mediav1connect.MediaService.On(cluster, node)
		},
		Roots:         library,
		Staging:       n.store,
		Controller:    side.Self(),
		Transfers:     receiver,
		Timings:       n.cfg.Timings.Grabs,
		Rand:          n.grabsRand,
		Changed:       grabs.ChangedEvent.Bind(n.hub),
		SearchChanged: grabs.SearchChangedEvent.Bind(n.hub),
		FilesChanged:  files.ChangedEvent.Bind(n.hub),
		BeforeStep:    n.env.Test.BeforeStep,
		Log:           n.rootLog.With("subsystem", "grabs"),
	})
	started := Half{Link: n.link, Dispatch: dispatch, Local: local, Library: lib.db, Controls: controls, Cluster: cluster}
	scheduler, err := tasks.Open(ctx, tasks.Config{
		Library: lib.db,
		Catalog: append(catalog(housekeeping.Config{
			Library:  lib.db,
			Nodes:    side,
			Identity: lib.ids,
			Tidy: func(node role.NodeID) housekeepingv1connect.HousekeepingServiceHandler {
				return housekeepingv1connect.HousekeepingService.On(cluster, node)
			},
		}), n.env.Test.tasks(started)...),
		Roster:     side,
		Zone:       n.zone(),
		Rand:       n.tasksRand,
		Hub:        n.hub,
		Changed:    tasks.ChangedEvent.Bind(n.hub),
		Progressed: tasks.ProgressEvent.Bind(n.hub),
		Log:        n.rootLog.With("subsystem", "tasks"),
	})
	if err != nil {
		return err
	}
	n.setHalf(&half{
		peers: side, dispatch: dispatch, holding: holding, systems: systems, nodes: side, controls: controls, cluster: cluster, works: tree, metadata: meta, indexers: registry,
		targets:   goals,
		roots:     library,
		files:     shelf,
		grabs:     pipeline,
		tasks:     scheduler,
		transfers: receiver,
	})
	n.claimed.Go("nodes", side.Run)
	n.claimed.Go("roots", library.Run)
	n.claimed.Go("grabs", pipeline.Run)
	n.claimed.Go("files", shelf.Run)
	n.claimed.Go("tasks", scheduler.Run)
	n.claimed.Go("metadata", meta.Run)
	n.claimed.Go("availability", metadata.NewAvailability(metadata.AvailabilityConfig{
		Library: lib.db, Local: local, Targets: goals, Hub: n.hub, Changed: works.ChangedEvent.Bind(n.hub),
		Now: n.today, Released: metadata.NoBacklog, Log: n.rootLog.With("subsystem", "availability"),
	}).Run)
	n.env.Test.started(started)
	return nil
}

// today is the time now in the Controller's zone.
func (n *Node) today() time.Time {
	if n.env.Test.Today != nil {
		return n.env.Test.Today().In(n.zone())
	}
	return time.Now().In(n.zone())
}

func (n *Node) zone() *time.Location {
	if n.env.Zone == nil {
		return time.Local
	}
	return n.env.Zone
}

// httpClient is the half's client for every host the Node reaches, dialing
// through the Node's network and trusting the OS roots.
func (n *Node) httpClient() *http.Client {
	transport := &http.Transport{
		DialContext:         n.env.Net.DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
	n.claimed.OnStop("http client", func(context.Context) error {
		transport.CloseIdleConnections()
		return nil
	})
	return &http.Client{Transport: transport}
}

// openHolder opens every Content Type's provider half over the Node's one
// call layer, each provider's Line registered once, and the providers'
// tests, in the order status lists them.
func (n *Node) openHolder(client *http.Client) metadata.HolderConfig {
	userAgent := "SlipStream/" + n.env.Version + " ( github.com/jatassi/SlipStream )"
	lines := metadata.NewLines(client, userAgent, n.cfg.Timings.Metadata, n.metadataRand)
	deps := contenttype.ProviderDeps{
		TMDB:  tmdb.New(lines, n.cfg.Endpoints.TMDB, n.cfg.Timings.TMDB),
		OMDb:  omdb.New(lines, n.cfg.Endpoints.OMDb, n.cfg.Timings.OMDb),
		Lines: lines,
	}
	sources := map[works.TypeID]metadata.Source{}
	for _, t := range n.env.Types.All() {
		deps.Log = n.rootLog.With("subsystem", "contenttype."+string(t.ID()))
		sources[t.ID()] = t.OpenProvider(deps, n.cfg.Endpoints.ContentTypes.section(t.ID()))
	}
	return metadata.HolderConfig{Types: sources, Testers: []metadata.Tester{deps.TMDB, deps.OMDb}, Lines: lines}
}

// openLibraries opens every Content Type's library half, on the
// Controller.
func (n *Node) openLibraries() map[works.TypeID]contenttype.Library {
	libraries := map[works.TypeID]contenttype.Library{}
	for _, t := range n.env.Types.All() {
		libraries[t.ID()] = t.OpenLibrary(contenttype.LibraryDeps{
			Region: metadata.Region,
			Log:    n.rootLog.With("subsystem", "contenttype."+string(t.ID())),
		})
	}
	return libraries
}

// startWorker starts the Worker half: the holder half over the link to its
// Controller, and call dispatch toward it.
func (n *Node) startWorker(ctx context.Context) error {
	side := n.pairing
	controller := side.Controller()
	front := serving.NewWorker(n.link, controller.Key, n.cfg.Timings.Serving, n.rootLog.With("subsystem", "serving"))
	holding, local, err := n.startHolding(ctx, role.OverLink(n.link, controller.Key, n.cfg.Timings.Link), []role.Held{role.HoldRun(role.Serving, front.Run, front.Release)})
	if err != nil {
		return err
	}
	dispatch := rpc.NewDispatch(n.link, n.key.Public(), controller.Key, holding.Holds)
	client := n.httpClient()
	searchv1connect.SearchingService.Mount(dispatch, indexers.NewHome(client, n.cfg.Timings.Indexers))
	metadatav1connect.MetadataService.Mount(dispatch, metadata.NewHolder(n.openHolder(client)))
	mediav1connect.MediaService.Mount(dispatch, media.NewService(n.env.Media, n.downloads, n.cfg.Timings.Media))
	notifyv1connect.NotifyingService.Mount(dispatch, notifying.Serve(n.cfg.Timings.Mail, n.env.Net.DialContext))
	storagev1connect.StorageService.Mount(dispatch, n.store)
	n.serveSystem(dispatch)
	housekeepingv1connect.HousekeepingService.Mount(dispatch, housekeeping.Service{Local: local})
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{peers: side, dispatch: dispatch, holding: holding, serving: front, source: transfers.NewSource(transfers.SourceConfig{
		Payloads:   n.downloads,
		Controller: controller.Key,
		Hooks:      n.env.Test.Transfers,
		Log:        n.rootLog.With("subsystem", "transfers"),
	})})
	n.link.Keep(controller)
	n.claimed.Go("pairing", side.Run)
	n.claimed.Go("serving", front.Follow)
	n.env.Test.started(Half{Link: n.link, Dispatch: dispatch, Local: local})
	return nil
}

// removed answers the Controller's removal of this Worker: the holder
// half forgets its cached intents, so its Roles stop and a later Join
// can't resume them, and the Node restarts Unclaimed.
func (n *Node) removed(ctx context.Context) {
	if h := n.half.Load(); h != nil {
		if err := h.holding.Reset(ctx); err != nil {
			n.log.Error("forget the removed Worker's Roles", "err", err.Error())
		}
	}
	if err := n.Restart(); err != nil {
		n.log.Error("restart the removed Worker", "err", err.Error())
	}
}

// startHolding opens local.db and runs the holder half toward upstream,
// holding the half's own Roles beside every Node's.
func (n *Node) startHolding(ctx context.Context, upstream role.Upstream, own []role.Held) (*role.Holding, *storage.DB[storage.Local], error) {
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), n.cfg.Timings.Storage, n.rootLog.With("subsystem", "storage"))
	if err != nil {
		return nil, nil, err
	}
	n.claimed.OnStop("local.db", func(context.Context) error { return local.Close() })
	holding, err := role.NewHolding(ctx, role.HoldingConfig{
		Local:    local,
		Upstream: upstream,
		Held:     slices.Concat(n.heldRoles(local), own, n.env.Test.held(local)),
		Timings:  n.cfg.Timings.Role,
		Rand:     n.roleRand,
		Log:      n.rootLog.With("subsystem", "role"),
	})
	if err != nil {
		return nil, nil, err
	}
	n.claimed.Go("role", holding.Run)
	return holding, local, nil
}

// setHalf records the half that started and reports the Node's host
// through it, then again at each probe.
func (n *Node) setHalf(h *half) {
	if !n.half.CompareAndSwap(nil, h) {
		panic("node: a second half started; a Node starts one, once")
	}
	n.reportHost()
	n.claimed.Go("probes", n.reprobe)
}

// serveSystem serves the Node's own system to its Controller, or to
// itself on a Controller.
func (n *Node) serveSystem(d *rpc.Dispatch) {
	systemv1connect.NodeSystemService.Mount(d, nodesystem.Serve(n.host, nodesystem.ServeTimings{
		LogPoll:       n.cfg.Timings.Link.LogPoll,
		LogArchiveTTL: n.cfg.Timings.Link.LogArchiveTTL,
	}))
}

// owners is each Role's owner: the feature that keeps its work.
func owners() []nodes.Owner {
	return []nodes.Owner{
		indexers.SearchingOwner,
		grabs.DownloadingOwner,
		roots.StorageOwner(files.RootContents{}),
		nodes.Unowned(role.Converting),
		nodes.Unowned(role.Subtitles),
		metadata.Owner,
		nodes.Unowned(role.MediaServerLink),
		notifying.Owner,
		nodes.Unowned(role.Relaying),
		nodes.Unowned(role.Serving),
	}
}

// heldRoles is every Role this binary runs, one held line each.
func (n *Node) heldRoles(local *storage.DB[storage.Local]) []role.Held {
	return []role.Held{
		downloading.Def.Hold(n.downloads.Run(local), n.downloads.Release),
		storing.Def.Hold(n.store.Run, n.store.Release),
	}
}

// claimPeers is how the link treats peers: as the half that started does,
// and refusing every one while the Node is Unclaimed.
type claimPeers struct{ n *Node }

func (p claimPeers) Admit(key link.Key) link.Admission {
	if h := p.n.half.Load(); h != nil {
		return h.peers.Admit(key)
	}
	return link.Refused{}
}

func (p claimPeers) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	if h := p.n.half.Load(); h != nil {
		return h.peers.Pair(ctx, peer, req)
	}
	return notAController(), nil
}

// claimStreams serves Transfer streams with the Worker half's source, and
// Serve streams from a Node that serves on the Controller, and resets
// them on any other Node.
type claimStreams struct{ n *Node }

func (c claimStreams) serve(peer link.Key, s link.Stream) {
	if h := c.n.half.Load(); h != nil && h.nodes != nil {
		if id, ok := h.nodes.NodeOf(peer); ok && h.nodes.Serves(role.Serving, id) {
			c.n.served.Offer(s)
			return
		}
	}
	_ = s.Reset()
}

// servingWorker is the Worker half's Serving, nil on any other Node.
func (n *Node) servingWorker() *serving.Worker {
	if h := n.half.Load(); h != nil {
		return h.serving
	}
	return nil
}

func (c claimStreams) transfer(peer link.Key, s link.Stream) {
	if h := c.n.half.Load(); h != nil && h.source != nil {
		h.source.Serve(peer, s)
		return
	}
	_ = s.Reset()
}

func notAController() *linkv1.PairResponse {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER}}}
}
