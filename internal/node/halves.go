package node

import (
	"context"
	"errors"
	"maps"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notifications"
	notificationproviders "github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifying"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1/convertingv1connect"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/housekeeping/v1/housekeepingv1connect"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/network/v1/networkv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1/storagev1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
	"github.com/jatassi/slipstream/internal/pb/slipstream/system/v1/systemv1connect"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/providers/gestdown"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/providers/subdl"
	"github.com/jatassi/slipstream/internal/providers/subsource"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/serving"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

// peers is how the link treats a peer under the Node's claim: whom it
// admits, and how it answers a Pair.
type peers interface {
	link.Trust
	Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error)
	Told(from link.Key, s link.Superseded)
}

// half is the claimed part of a running Node. Each Node starts at most one,
// from Unclaimed or at start, and never restarts it in process.
type half struct {
	backups *backups.Backups
	// self is the Node's id in its Controller's library.
	self     role.NodeID
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
	trackers *trackers.Trackers
	targets  *targets.Targets
	roots    *roots.Roots
	files    *files.Files
	naming   *naming.Naming
	library  *library.Library
	imports  *imports.Imports
	grabs    *grabs.Grabs
	clients  *downloadclients.Clients
	// reconciling is Reconciling after a restore, on a Controller.
	reconciling *reconciling.Service
	// mediaServers is the Media Servers' operations, on a Controller.
	mediaServers mediaservers.Service

	// mediaServersHealth is the Media Servers' Health category, on a
	// Controller.
	mediaServersHealth health.Source

	// reachability is Remote access, on a Controller.
	reachability *reachability.Reachability

	// notifications is the Admin's notification connections and the bell,
	// on a Controller.
	notifications *notifications.Service

	// calendar reads release dates in a span of days, on a Controller.
	calendar *calendar.Calendar

	// home reads Home and keeps its setting, on a Controller.
	home *home.Service

	// history lists what happened to the library, on a Controller.
	history *history.Service

	// subtitles is the library's wanted subtitles, on a Controller.
	subtitles *subtitles.Subtitles

	tasks    *tasks.Scheduler
	health   *health.Health
	firstRun *firstrun.FirstRun
	// transfers judges every Transfer, on a Controller, and copier sends
	// and receives this Node's, on every Node.
	transfers *transfers.Transfers
	copier    *transfers.Node
	// moves drives every Move, on a Controller.
	moves *moves.Moves
	// conversions queues, places and lands every Conversion, on a
	// Controller.
	conversions *conversions.Conversions
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
	client := n.httpClient(n.env.Net.DialContext)
	holder := n.openHolder(client)
	opened := n.openLibraries()
	var (
		downloads      *role.Control[*dlv1.Intent, *dlv1.Facts, *dlv1.Report]
		transferring   *role.Control[*tv1.Intent, *tv1.Facts, *tv1.Result]
		linked         *role.Control[*msv1.Intent, *msv1.Facts, *msv1.Report]
		side           *nodes.Service
		cluster        *rpc.Cluster
		homes          *roots.Roots
		seeding        *trackers.Trackers
		mover          *moves.Moves
		pipeline       *grabs.Grabs
		scheduler      *tasks.Scheduler
		meta           *metadata.Metadata
		found          *imports.Imports
		shelf          *files.Files
		searchQueue    = &searches.Late{}
		kept           *backups.Backups
		goals          *targets.Targets
		converts       *conversions.Conversions
		conversionRuns *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result]
	)
	var keeping entry.Recorder
	keeper := reconciling.New(reconciling.Config{
		Library: lib.db,
		Self:    func() role.NodeID { return side.Self() },
		History: &keeping,
		Touch:   func(ids ...role.NodeID) { downloads.Touch(ids...) },
		Metainfo: func(ctx context.Context, node role.NodeID, hash bencode.InfoHash, client *downloadclients.ClientID) ([]byte, error) {
			return metainfoOn(ctx, cluster, node, hash, client)
		},
		Page: n.cfg.Timings.Downloading.HoldingsPage,
		Max:  n.cfg.Timings.Downloading.HoldingsMax,
		Rehome: func(ctx context.Context, by identity.Admin, id roots.ID, h roots.Rehome) (roots.View, error) {
			return homes.Rehome(ctx, by, id, h)
		},
		Scan:  func(ctx context.Context, ids []roots.ID) error { return found.ScanRoots(ctx, ids) },
		Ready: func(ctx context.Context) ([]roots.Root, error) { return homes.Ready(ctx) },
		Hub:   n.hub,
		Ending: func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return pipeline.DecideFetching(ctx, tx)
		},
		Ended: func(ctx context.Context) {
			pipeline.Reconciled(ctx)
			found.Touch()
			shelf.Poke()
			scheduler.Reconciled()
		},
		RootWords: func(ctx context.Context, r storage.Reader[storage.Library]) (map[roots.ID]string, error) {
			return homes.WordsIn(ctx, r)
		},
		Roster: openedRoster{&side},
		Look: func(ctx context.Context, node role.NodeID, folder string, files []downloadclients.LookFile) ([]bool, error) {
			return lookOn(ctx, cluster, node, folder, files)
		},
		Forgot:  func() { pipeline.Rerouted() },
		Changed: reconciling.ChangedEvent.Bind(n.hub),
		Log:     n.rootLog.With("subsystem", "reconciling"),
	})
	var tree *works.Works
	clients := downloadclients.New(downloadclients.Config{
		Library:      lib.db,
		Touch:        func(nodes ...role.NodeID) { downloads.Touch(nodes...) },
		Changed:      downloadclients.ChangedEvent.Bind(n.hub),
		CardsChanged: downloadclients.ClientsChangedEvent.Bind(n.hub),
		Volumes:      roots.VolumesChangedEvent.Bind(n.hub),
		Rerouted:     func() { pipeline.Rerouted() },
		Facts:        func(id role.NodeID) (*dlv1.Facts, bool) { return downloads.Facts(id) },
		Remote: func(id role.NodeID) downloadingv1connect.DownloadingServiceHandler {
			return downloadingv1connect.DownloadingService.On(cluster, id)
		},
		Sealer:    lib.sealer,
		Roster:    openedRoster{&side},
		Uses:      imports.Uses{TorrentUses: grabs.TorrentUses{}},
		Blocklist: grabs.BlocklistOn{Reconciling: reconciling.Blocked, Changes: grabs.BlocklistChangedEvent.Bind(n.hub)},
		Seeding:   lateSeeder{&seeding},
		Moving:    lateMoves{&mover},
		Types:     lateTypes{&tree},
		Live: downloadclients.LiveConfig{
			Watched: func() []role.NodeID { return side.Who(role.Downloading).InStep() },
			Event:   downloadclients.LiveEvent.Bind(n.hub),
			Sample:  n.cfg.Timings.Downloading.LiveSample,
		},
		Hub:          n.hub,
		Keeps:        keeper,
		HoldingsPage: n.cfg.Timings.Downloading.HoldingsPage,
		HoldingsMax:  n.cfg.Timings.Downloading.HoldingsMax,
		Log:          n.rootLog.With("subsystem", "downloadclients"),
	})
	seeding = trackers.New(trackers.Config{
		Library:         lib.db,
		Clients:         clients,
		Grabs:           grabs.TrackerFacts{},
		Roster:          openedRoster{&side},
		Routing:         trackers.Routing{Roster: openedRoster{&side}},
		Rerouted:        func() { pipeline.Rerouted() },
		Soon:            func() { scheduler.Soon(trackers.TaskID) },
		Reconciling:     reconciling.Blocked,
		Changed:         trackers.SeedingChangedEvent.Bind(n.hub),
		TrackersChanged: indexers.TrackersEvent.Bind(n.hub),
		Moving:          lateMoves{&mover},
	})
	downloads = downloading.Def.Serve(downloadclients.Derive{Keeps: keeper, HoldingsPage: n.cfg.Timings.Downloading.HoldingsPage}.Intent, clients.Apply)
	stored := storing.Def.Serve(roots.StorageIntent(files.RootContents{}), roots.ApplyStorage)
	copies := transfers.New(transfers.ControllerConfig{
		Library: lib.db,
		Touch:   func(nodes ...role.NodeID) { transferring.Touch(nodes...) },
		Facts:   func(id role.NodeID) (*tv1.Facts, bool) { return transferring.Facts(id) },
		Board:   func() transfers.Board { return transfers.BoardOf(side.Board()) },
		Destinations: func(ctx context.Context, r storage.Reader[storage.Library], ids []roots.ID) (map[roots.ID]roots.Destination, error) {
			return roots.DestinationsIn(ctx, r, ids, stored, side)
		},
		RootsMoved: func() <-chan struct{} { return homes.Watch() },
		Readiness: func(ctx context.Context, rd storage.Reader[storage.Library], node role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error) {
			out, err := clients.ReadinessIn(ctx, rd, node, refs)
			if err != nil {
				return nil, err
			}
			library, err := roots.LibraryReadiness(ctx, rd, refs)
			maps.Copy(out, library)
			return out, err
		},
		Moved:         clients.Moved,
		LowWait:       backups.CopyWaitLimitIn,
		Paused:        whileReconciling,
		Folders:       func(id role.NodeID) []transfers.BackupFolder { return kept.Folders(id) },
		FoldersNoted:  func(id role.NodeID) { kept.FoldersNoted(id) },
		Sample:        func(ctx context.Context, id role.NodeID) error { return homes.Sample(ctx, id) },
		ScratchVolume: scratchVolume,
		Hub:           n.hub,
		Changed:       transfers.ChangedEvent.Bind(n.hub),
		Timings:       n.cfg.Timings.Transfers,
		Log:           n.rootLog.With("subsystem", "transfers"),
	})
	transferring = transfers.Def.Serve(copies.Derive, copies.Apply)
	servers := mediaservers.New(mediaservers.Config{
		Library: lib.db,
		Touched: func() { converts.TouchHolders() },
		Sealer:  lib.sealer,
		Roster:  openedRoster{&side},
		Kinds:   mediaserverlink.Kinds,
		HTTP:    client,
		Timings: n.cfg.Timings.MediaServers,
		PlexTV:  n.cfg.Endpoints.PlexTV,
		Link: func(node role.NodeID) mediaserverv1connect.MediaServerLinkServiceHandler {
			return mediaserverv1connect.MediaServerLinkService.On(cluster, node)
		},
		FileIDs:    files.IDsAt,
		FilePlaces: files.PlacesOf,
		Played:     playbackReads{&goals}.played,
		Siblings:   playbackReads{&goals}.siblings,
		Targets:    targets.All,
		Batch:      n.cfg.Timings.Housekeeping.PruneBatch,
		WorkFiles: func(ctx context.Context, r storage.Reader[storage.Library], work works.WorkID) ([]files.File, error) {
			return files.LeftIn(ctx, r, []works.WorkID{work})
		},
		RootWords: func(ctx context.Context, r storage.Reader[storage.Library]) (map[roots.ID]string, error) {
			return homes.WordsIn(ctx, r)
		},
		RootsChanged: func(ctx context.Context) <-chan struct{} { return n.hub.Watch(ctx, roots.ChangedEvent) },
		Tasks:        lateTasks{&scheduler},
		Events: mediaservers.Events{
			Changed:         mediaservers.ChangedEvent.Bind(n.hub),
			SessionsChanged: mediaservers.SessionsChangedEvent.Bind(n.hub),
			PlaysChanged:    mediaservers.PlaysChangedEvent.Bind(n.hub),
			Unhealthy:       mediaservers.UnhealthyEvent.Bind(n.hub),
			Recovered:       mediaservers.RecoveredEvent.Bind(n.hub),
			ItemsFound:      mediaservers.ItemsFoundEvent.Bind(n.hub),
			AccessChanged:   mediaservers.AccessChangedEvent.Bind(n.hub),
			MyAccessChanged: mediaservers.MyAccessChangedEvent.Bind(n.hub),
		},
		Touch: func(nodes ...role.NodeID) { linked.Touch(nodes...) },
		Facts: func(node role.NodeID) (*msv1.Facts, bool) { return linked.Facts(node) },
		Zone:  n.zone(),
		Log:   n.rootLog.With("subsystem", "mediaservers"),
		Admins: func(ctx context.Context) ([]mail.Address, error) {
			return identity.EnabledAdminAddresses(ctx, lib.db)
		},
		Mail:      lib.mail,
		Outbox:    lib.outbox,
		PlexFacts: plextv.Recorded,
	})
	linked = mediaserverlink.Def.Serve(servers.Intent, servers.Apply)
	lib.ids.SetGranter(servers.Granter())
	reach := reachability.New(reachability.Config{
		Library: lib.db,
		Hub:     n.hub,
		Log:     n.rootLog.With("subsystem", "reachability"),
		Roster:  reachRoster{openedRoster{&side}},
		Tasks:   lateTasks{&scheduler},
		Network: func(id role.NodeID) networkv1connect.NodeNetworkServiceHandler {
			return networkv1connect.NodeNetworkService.On(cluster, id)
		},
		MediaServers: servers,
		Rules:        netmodel.RulesOf(n.cfg.Network),
		LinkSource:   n.linkSource(),
		WebPort:      n.webPort,
		Timings:      n.cfg.Timings.Reachability,
		Zone:         n.zone(),
		Changed:      reachability.ChangedEvent.Bind(n.hub),
		Now:          time.Now,
	})
	network := netmodel.Def.Serve(reach.Intent, reach.Apply)
	reach.Serve(network)
	cat := entry.NewCatalog()
	n.notifications = notifications.New(notifications.Config{
		Library:   lib.db,
		Catalog:   cat,
		Sealer:    lib.sealer,
		Providers: notificationproviders.Table,
		Courier:   lateCourier{&cluster},
		Roster:    openedRoster{&side},
		Mail:      lib.mail,
		Artwork:   lateArtwork{&meta},
		Endpoints: n.cfg.Endpoints.Notify,
		Changed:   notifications.ChangedEvent.Bind(n.hub),
		Seen:      notifications.SeenEvent.Bind(n.hub),
		Timings:   n.cfg.Timings.Notifications,
		Rand:      n.notificationsRand,
		Zone:      n.zone(),
		Log:       n.rootLog.With("subsystem", "notifications"),
	})
	n.history = history.New(history.Config{
		Library: lib.db, Catalog: cat, Outbox: n.notifications, Changed: history.ChangedEvent.Bind(n.hub),
		Batch: n.cfg.Timings.Housekeeping.PruneBatch, Log: n.rootLog.With("subsystem", "history"),
	})
	keeping = n.history.Recorder("reconciling", reconciling.Entries())
	kept, err = backups.Open(ctx, backups.Config{
		Library: lib.db,
		Keys:    lib.keys,
		Engine:  n.backups,
		Start: func(ctx context.Context, st tasks.Start) (tasks.Started, error) {
			return scheduler.Start(ctx, st)
		},
		Reconciling: whileReconciling,
		History:     n.history.Recorder("backups", backups.Entries()),
		Log:         n.rootLog.With("subsystem", "backups"),
		Transfers:   copies,
		Self:        func() role.NodeID { return side.Self() },
		Check: func(ctx context.Context, admin identity.Admin, node role.NodeID, folder string) (roots.Verdict, error) {
			return homes.Check(ctx, admin, node, folder, storing.PurposeBackups, nil)
		},
		Tasks: lateTasks{&scheduler},
		Roster: func(id role.NodeID) (string, bool) {
			v, _ := side.Board().Node(id)
			return v.Name, v.Standing == nodes.InStep
		},
		Changed: backups.ChangedEvent.Bind(n.hub),
		Zone:    n.zone(),
	})
	if err != nil {
		return err
	}
	arrivals := roots.NewArrivals()
	fenced := nodes.FencedEvent.Bind(n.hub)
	var systems *nodesystem.Nodes
	conversionRuns = converting.Def.Serve(
		func(ctx context.Context, r storage.Reader[storage.Library], id role.NodeID) (*cv1.Intent, error) {
			return converts.Derive(ctx, r, id)
		},
		func(ctx context.Context, tx *storage.Tx[storage.Library], id role.NodeID, res *cv1.Result) error {
			return converts.Apply(ctx, tx, id, res)
		},
	)
	side, err = nodes.Open(ctx, nodes.Config{
		Library: lib.db,
		Key:     n.key.Public(),
		Name:    n.cfg.Name,
		Node:    n.info,
		Link:    n.link,
		Changed: nodes.ChangedEvent.Bind(n.hub),
		Traffic: nodes.TrafficEvent.Bind(n.hub),

		SettingsChanged: nodes.SettingsChangedEvent.Bind(n.hub),
		Served:          append([]role.Served{downloads, stored, transferring, linked, network, conversionRuns}, n.env.Test.Served...),
		Owners:          owners(clients, servers, n.notifications),
		Kept:            grabs.Kept,
		Torrents:        clients,
		Storage: func(ctx context.Context, r storage.Reader[storage.Library], id role.NodeID) (nodes.StorageFigures, error) {
			return homes.Figures(ctx, r, id)
		},
		Television: servers.Television,
		Facts: func(id role.NodeID, scope string) {
			arrivals.Note(id, scope)
			clients.Arrived(id, scope)
			copies.Noted(id, scope)
			servers.Noted(id, scope)
			reach.Noted(id, scope)
			converts.Noted(id, scope)
		},
		Applied: func(id role.NodeID, scope string) {
			clients.Applied(id, scope)
			keeper.Applied(id, scope)
			copies.Applied(id, scope)
			servers.Applied(id, scope)
		},
		Reach: func(ctx context.Context, id role.NodeID, t nodesystem.Target) (nodesystem.Reached, error) {
			return systems.Reach(ctx, id, t)
		},
		Probe:  func(ctx context.Context, id role.NodeID) ([]role.ProbeResult, error) { return systems.Probe(ctx, id) },
		Always: n.env.Test.Always,
		Fence:  kept,
		Fenced: func(ctx context.Context, superseded bool, ids []role.NodeID) {
			downloads.Touch(ids...)
			fenced.Publish(ctx, nodes.Fenced{Superseded: superseded})
		},
		Paired:  reconciling.Paired,
		Pending: reconciling.Pending,
		Timings: n.cfg.Timings.Nodes,
		Stall:   n.cfg.Timings.Role.Stall,
		Log:     n.rootLog.With("subsystem", "nodes"),
	})
	if err != nil {
		return err
	}
	controls := side.Controls()
	self := n.key.Public()
	var holding *role.Holding
	dispatch := rpc.NewDispatch(n.link, self, self, func(r role.Name) bool { return holding.Holds(r) })
	cluster = rpc.NewCluster(dispatch, side)
	n.mediaLink = n.newMediaLink()
	holding, local, copier, err := n.startHolding(ctx, side.Self(), role.InProcess(controls), dispatch, nil, nil)
	if err != nil {
		return err
	}
	searchv1connect.SearchingService.Mount(dispatch, n.searchingService())
	metadatav1connect.MetadataService.Mount(dispatch, metadata.NewHolder(holder))
	subtitlesv1connect.SubtitlesService.Mount(dispatch, n.subtitlesHolder(client))
	mediav1connect.MediaService.Mount(dispatch, media.NewService(n.env.Media, n.downloading, n.store.Holds, n.store.LibraryFile, n.cfg.Timings.Media))
	notifyv1connect.NotifyingService.Mount(dispatch, notifying.Serve(n.cfg.Timings.Mail, n.env.Net.DialContext, n.env.Scripts))
	if err := n.mountSidecars(ctx, dispatch, local); err != nil {
		return err
	}
	storagev1connect.StorageService.Mount(dispatch, n.store)
	storagev1connect.FolderService.Mount(dispatch, n.store)
	downloadingv1connect.DownloadingService.Mount(dispatch, n.downloading)
	mediaserverv1connect.MediaServerLinkService.Mount(dispatch, n.mediaLink.Service())
	convertingv1connect.ConvertingService.Mount(dispatch, converting.Service(n.proofs))
	n.serveSystem(dispatch)
	networkv1connect.NodeNetworkService.Mount(dispatch, netfacts.NewService(n.network))
	providers := make([]calls.Provider, len(holder.Testers))
	for i, t := range holder.Testers {
		providers[i] = t.Provider()
	}
	typeChanged := works.TypeChangedEvent.Bind(n.hub)
	meta, err = metadata.Open(ctx, metadata.Config{
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
		Sources:     holder.Sources(),
	})
	if err != nil {
		return err
	}
	tree, err = works.Open(ctx, works.Config{
		Library:      lib.db,
		Catalogs:     contenttype.Project(opened, func(t contenttype.Library) works.Catalog { return t }),
		Source:       meta,
		Artwork:      meta,
		Timings:      n.cfg.Timings.Works,
		Changed:      works.ChangedEvent.Bind(n.hub),
		Removed:      works.RemovedEvent.Bind(n.hub),
		TypesChanged: works.ContentTypesChangedEvent.Bind(n.hub),
		TypeChanged:  typeChanged,
		Log:          n.rootLog.With("subsystem", "works"),
		ItemsAdded: func(ctx context.Context, tx *storage.Tx[storage.Library], work works.WorkID, items []works.ItemID) error {
			return searchQueue.Queue(ctx, tx, searches.Refresh, searches.Ask{Work: work, Items: items})
		},
	})
	if err != nil {
		return err
	}
	downloadingv1connect.DownloadingControlService.Mount(cluster, downloadclients.ControlService{Library: lib.db, Sealer: lib.sealer, Nodes: side, Types: tree})
	mediaserverv1connect.MediaServerControlService.Mount(cluster, mediaservers.ControlService{Library: lib.db, Sealer: lib.sealer, Nodes: side})
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
		Searching: func(node role.NodeID) searching.Remote {
			return searching.RemoteOf(searchv1connect.SearchingService.On(cluster, node))
		},
		Ports:      n.ports,
		Types:      contenttype.Project(opened, func(t contenttype.Library) indexers.Facet { return t }),
		Works:      tree,
		Changed:    indexers.ChangedEvent.Bind(n.hub),
		Status:     indexers.StatusChangedEvent.Bind(n.hub),
		Catalogs:   indexers.CatalogChangedEvent.Bind(n.hub),
		Trackers:   indexers.TrackersEvent.Bind(n.hub),
		Settings:   indexers.SettingsChangedEvent.Bind(n.hub),
		RSSChanged: indexers.RSSChangedEvent.Bind(n.hub),
		Repoints:   []indexers.Repoint{grabs.RepointTracker, downloadclients.RepointTracker, trackers.RepointTracker},
		HTTP:       client,
		Endpoints:  n.cfg.Endpoints.Indexers,
		CatalogNext: func(ctx context.Context) (time.Time, error) {
			return scheduler.Next(ctx, indexers.CatalogRefreshTaskID)
		},
		Timings: n.cfg.Timings.Indexers,
		JobCap:  n.cfg.Timings.Searching.JobCap,
		RSS: func(ctx context.Context, r storage.Reader[storage.Library]) (time.Duration, bool, error) {
			return pipeline.RSSIn(ctx, r)
		},
		Zone: n.zone(),
		Log:  n.rootLog.With("subsystem", "indexers"),
	})
	goals, err = targets.Open(ctx, targets.Config{
		Library:   lib.db,
		Works:     tree,
		Types:     contenttype.Project(opened, func(t contenttype.Library) targets.Facet { return t }),
		Subtitled: func(t works.TypeID) bool { return len(opened[t].Subtitled()) > 0 },
		Changed:   targets.ChangedEvent.Bind(n.hub),
		Log:       n.rootLog.With("subsystem", "targets"),
		Formats:   n.cfg.Timings.Formats,
	})
	if err != nil {
		return err
	}
	stores := storing.Remote(func(id role.NodeID) storagev1connect.StorageServiceHandler {
		return storagev1connect.StorageService.On(cluster, id)
	})
	homes, err = roots.Open(ctx, roots.Config{
		Library: lib.db,
		Types:   types,
		Roster:  side,
		Storage: stored,
		Remote:  stores,
		Folders: func(id role.NodeID) storagev1connect.FolderServiceHandler {
			return storagev1connect.FolderService.On(cluster, id)
		},
		Arrivals:    arrivals,
		Contents:    files.RootContents{},
		Inflight:    imports.RootImports{},
		Busy:        rootBusy,
		Reconciling: reconciling.Blocked,
		Rehomed:     servers,
		Videos:      imports.Videos,
		Changed:     roots.ChangedEvent.Bind(n.hub),
		Volumes:     roots.VolumesChangedEvent.Bind(n.hub),
		Cadence:     clients.CadenceChanged,
		Names:       side.Names,
		Describe:    tree.Describe,
		Log:         n.rootLog.With("subsystem", "roots"),
	})
	if err != nil {
		return err
	}
	router := downloadclients.Router{Roster: side, Routes: trackers.Routing{Roster: side}}
	names := naming.New(naming.Config{
		Library: lib.db, Facets: contenttype.Project(opened, func(t contenttype.Library) naming.Facet { return t }),
		Describe: tree.Describe, Budget: roots.SmallestBudget, Changed: naming.ChangedEvent.Bind(n.hub),
	})
	subs := &lateSubtitles{Subtitles: files.NoSubtitles{}}
	poke := conversions.NewPoke()
	shelf, err = files.Open(ctx, files.Config{
		Queue:   searchQueue,
		Library: lib.db, Targets: goals, Grabs: imports.Cells{Grabs: grabs.GrabCells{InStep: side.InStep, Router: router}}, Transfers: transfers.LiveIn, MediaServers: servers, Subtitles: subs,
		Naming:      names,
		Names:       contenttype.Project(opened, func(t contenttype.Library) release.Names { return t }),
		Today:       n.today,
		Disk:        homes,
		Changed:     files.ChangedEvent.Bind(n.hub),
		Moved:       files.MovedEvent.Bind(n.hub),
		Timings:     n.cfg.Timings.Files,
		Rand:        n.filesRand,
		Reconciling: reconciling.Blocked,
		Moves:       moves.FilesHooks{},
		Conversions: conversions.Seam{Poke: poke},
		Homes:       homes,
		Log:         n.rootLog.With("subsystem", "files"),
	})
	if err != nil {
		return err
	}
	subtitler, err := subtitles.Open(ctx, subtitles.Config{
		Library: lib.db, Files: shelf, Targets: goals, Roots: homes,
		Types: contenttype.Project(opened, func(t contenttype.Library) subtitles.Facet { return t }),
		Storage: func(id role.NodeID) storing.SidecarCalls {
			return storing.SidecarCallsOf(storagev1connect.SidecarService.On(cluster, id))
		},
		Holders: func(id role.NodeID) subtitling.Remote {
			return subtitling.NewRemote(subtitlesv1connect.SubtitlesService.On(cluster, id))
		},
		Videos:    imports.Videos,
		Providers: n.subtitleSpecs(),

		MediaServers: servers,
		Works:        tree,
		FetchCap:     n.cfg.Timings.Subtitling.FetchCap,

		Tasks:       lateTasks{&scheduler},
		GapCap:      subtitles.NoGapCap,
		Roster:      side,
		Reconciling: reconciling.Blocked,
		History:     n.history.Recorder("subtitles", subtitles.Entries()),
		Timings:     n.cfg.Timings.Subtitles,
		Clock:       time.Now,
		Rand:        n.subtitlesRand,
		Log:         n.rootLog.With("subsystem", "subtitles"),

		Sealer:           lib.sealer,
		Describe:         tree.Describe,
		Changed:          subtitles.ChangedEvent.Bind(n.hub),
		ProvidersChanged: subtitles.ProvidersChangedEvent.Bind(n.hub),
		Attempt:          subtitles.AttemptEvent.Bind(n.hub),
	})
	if err != nil {
		return err
	}
	subs.Subtitles = subtitler
	dates := calendar.New(calendar.Config{
		Library: lib.db, Types: contenttype.Project(opened, func(t contenttype.Library) calendar.Facet { return t }),
		Works: tree, Files: shelf, Today: n.today,
	})
	pipeline = grabs.New(grabs.Config{
		Library:          lib.db,
		Works:            tree,
		Targets:          goals,
		Files:            shelf,
		Indexers:         registry,
		Sealer:           lib.sealer,
		Terms:            contenttype.Project(opened, func(t contenttype.Library) indexers.Facet { return t }),
		Names:            contenttype.Project(opened, func(t contenttype.Library) release.Names { return t }),
		Roster:           side,
		Downloads:        clients,
		Router:           router,
		Roots:            homes,
		Imports:          imports.GrabImports{},
		Timings:          n.cfg.Timings.Grabs,
		Rand:             n.grabsRand,
		Changed:          grabs.ChangedEvent.Bind(n.hub),
		SearchChanged:    grabs.SearchChangedEvent.Bind(n.hub),
		FilesChanged:     files.ChangedEvent.Bind(n.hub),
		BlocklistChanged: grabs.BlocklistChangedEvent.Bind(n.hub),
		Reconciling:      reconciling.Blocked,
		BeforeStep:       n.env.Test.BeforeStep,
		Tasks:            lateTasks{&scheduler},
		GapCap:           grabs.NoGapCap,
		QueueChanged:     grabs.QueueChangedEvent.Bind(n.hub),
		Clock:            time.Now,
		Today:            n.today,
		History:          n.history.Recorder("grabs", grabs.Entries()),
		Log:              n.rootLog.With("subsystem", "grabs"),
	})
	searchQueue.Bind(pipeline)
	var composer *library.Library
	probes := func(node role.NodeID) mediav1connect.MediaServiceHandler {
		return mediav1connect.MediaService.On(cluster, node)
	}
	found = imports.New(imports.Config{
		Library: lib.db,
		Remote:  stores,
		Roots:   homes,
		Files:   shelf,
		Works:   tree,
		Targets: goals,
		Facets:  contenttype.Project(opened, func(t contenttype.Library) scanning.Facet { return t }),
		Add: func(ctx context.Context, by identity.Admin, a library.Adding) (library.Added, error) {
			return composer.Add(ctx, by, a)
		},
		Start: func(ctx context.Context, st tasks.Start) (tasks.Started, error) {
			return scheduler.Start(ctx, st)
		},
		Steps: func(task tasks.TaskID) (tasks.RunID, map[tasks.UnitKey]tasks.Step, bool) {
			return scheduler.Steps(task)
		},
		Hub:          n.hub,
		Changed:      imports.ChangedEvent.Bind(n.hub),
		FilesChanged: files.ChangedEvent.Bind(n.hub),
		Probes:       probes,
		Transfers:    copies,
		OtherCopies:  otherCopies(&converts),
		Grabs:        pipeline,
		Timings:      imports.Timings{RetryMin: n.cfg.Timings.Grabs.RetryMin, RetryMax: n.cfg.Timings.Grabs.RetryMax},
		Rand:         n.importsRand,
		BeforeStep:   n.env.Test.BeforeScanStep,
		BeforeImport: n.env.Test.BeforeImport,
		History:      n.history.Recorder("imports", imports.Entries()),
		Subtitles:    subtitler,
		Reconciling:  keeper,
		Log:          n.rootLog.With("subsystem", "imports"),
	})
	pipeline.Register(found.Importer())
	mover = moves.New(moves.Config{
		Library: lib.db, Works: tree, Files: shelf, Roots: homes, Disk: homes, Imports: found, Transfers: copies, MediaServers: servers,
		Destinations: func(ctx context.Context, r storage.Reader[storage.Library], ids []roots.ID) (map[roots.ID]roots.Destination, error) {
			return roots.DestinationsIn(ctx, r, ids, stored, side)
		},
		Writing: writing, Roster: side, Videos: imports.Videos, Reconciling: reconciling.Blocked, Hub: n.hub,
		Changed: moves.ChangedEvent.Bind(n.hub), FilesChanged: files.ChangedEvent.Bind(n.hub),
		Touch:   func(nodes ...role.NodeID) { stored.Touch(nodes...) },
		Clients: clients,
		Downloading: func(id role.NodeID) downloadingv1connect.DownloadingServiceHandler {
			return downloadingv1connect.DownloadingService.On(cluster, id)
		},
		Repoint: grabs.RepointTorrentIn, CopyGoalRun: trackers.CopyGoalRunIn, Reroute: trackers.RerouteIn,
		Recorder: n.history.Recorder("moves", moves.Entries()), Timings: n.cfg.Timings.Moves, BeforeStep: n.env.Test.BeforeMoveStep,
		Clock: time.Now, Log: n.rootLog.With("subsystem", "moves"),
	})
	copies.Register(mover.Purpose())
	converts = conversions.New(conversions.Config{
		Library:   lib.db,
		Files:     shelf,
		Targets:   goals,
		Transfers: copies,
		Board:     side.Board,
		Control:   func() *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result] { return conversionRuns },
		Converting: func(node role.NodeID) convertingv1connect.ConvertingServiceClient {
			return convertingv1connect.ConvertingService.On(cluster, node)
		},
		Probes:        probes,
		RootRoom:      found.CopyRoom,
		TranscodingOn: servers.TranscodingOn,
		History:       n.history.Recorder("conversions", conversions.Entries()),
		Changed:       conversions.ChangedEvent.Bind(n.hub),
		Live:          conversions.LiveEvent.Bind(n.hub),
		Hub:           n.hub,
		Poke:          poke,
		Reconciling:   reconciling.Blocked,
		Zone:          n.zone,
		Timings:       n.cfg.Timings.Conversions,
		Run:           n.cfg.Timings.Converting,
		Clock:         time.Now,
		Log:           n.rootLog.With("subsystem", "conversions"),
	})
	front := home.New(home.Config{
		Library: lib.db, Moves: mover, Grabs: pipeline, Imports: found, Works: tree, Files: shelf, Clients: clients,
		Changed: home.SettingsChangedEvent.Bind(n.hub),
	})
	if err := found.Carry(ctx); err != nil {
		return err
	}
	composer = library.New(library.Config{Library: lib.db, Works: tree, Targets: goals, Roots: homes, Files: shelf, Imports: found, Leaver: found, Queue: searchQueue, Moves: mover.Hooks().Library,
		History: n.history.Recorder("library", library.Entries()), Subtitles: subtitler,
	})
	cat.Seal()

	started := Half{Link: n.link, Dispatch: dispatch, Local: local, Library: lib.db, Controls: controls, Cluster: cluster, Clients: clients, Imports: found, Transfers: copies, Moves: mover}
	scheduler, err = tasks.Open(ctx, tasks.Config{
		Library: lib.db,
		Catalog: append(catalog(housekeeping.Config{
			Library:  lib.db,
			Nodes:    side,
			Identity: lib.ids,
			Tidy: func(node role.NodeID) housekeepingv1connect.HousekeepingServiceHandler {
				return housekeepingv1connect.HousekeepingService.On(cluster, node)
			},
			Keep:  n.history,
			Batch: n.cfg.Timings.Housekeeping.PruneBatch,
			Steps: append(housekeepingSteps(registry, servers, n.history, pipeline, found, converts, shelf, subtitler), moves.Retention(lib.db)),
			Reconciling: func(ctx context.Context) (paused bool, err error) {
				err = lib.db.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
					paused, err = reconciling.Blocked(ctx, r)
					return err
				})
				return paused, err
			},
		}, metadata.RefreshConfig{
			Library:  lib.db,
			Metadata: meta,
			Works:    tree,
			Targets:  goals,
			Types:    contenttype.Project(opened, func(t contenttype.Library) metadata.Refreshing { return t }),
			Today:    n.today,
			Log:      n.rootLog.With("subsystem", "metadata-refresh"),
		}, found, kept), append(append(append(registry.Tasks(), seeding.Task(), servers.RenewalTask(), subtitler.Task(), reach.Task()), pipeline.Tasks()...), n.env.Test.tasks(started)...)...),
		Roster:      side,
		Zone:        n.zone(),
		Rand:        n.tasksRand,
		Hub:         n.hub,
		Changed:     tasks.ChangedEvent.Bind(n.hub),
		Progressed:  tasks.ProgressEvent.Bind(n.hub),
		Reconciling: reconciling.Blocked,
		Log:         n.rootLog.With("subsystem", "tasks"),
	})
	if err != nil {
		return err
	}
	h := &half{
		self: side.Self(), peers: side, dispatch: dispatch, holding: holding, systems: systems, nodes: side, controls: controls, cluster: cluster, works: tree, metadata: meta, indexers: registry,
		trackers: seeding, reconciling: keeper,
		targets: goals,
		roots:   homes,
		files:   shelf,
		naming:  names,
		library: composer,
		imports: found,
		grabs:   pipeline,
		clients: clients,
		tasks:   scheduler,
		backups: kept,

		mediaServers: servers.Service(),

		mediaServersHealth: servers.HealthSource(),

		reachability: reach,

		notifications: n.notifications,

		calendar: dates,

		home: front,

		history: n.history,

		subtitles: subtitler,

		transfers:   copies,
		copier:      copier,
		moves:       mover,
		conversions: converts,
	}
	if h.health, err = n.openHealth(ctx, lib.db, h); err != nil {
		return err
	}
	h.firstRun = n.openFirstRun(lib, h)
	n.setHalf(h)
	n.claimed.Go("nodes", side.Run)
	n.claimed.Go("roots", homes.Run)
	n.claimed.Go("indexers", registry.Run)
	n.claimed.Go("grabs", pipeline.Run)
	n.claimed.Go("torrent clients", clients.Run)
	n.claimed.Go("reconciling", keeper.Run)
	n.claimed.Go("transfers", copies.Run)
	n.claimed.Go("backup copies", kept.Run)
	n.claimed.Go("media servers", servers.Run)
	n.claimed.Go("reachability", reach.Run)
	n.claimed.Go("files", shelf.Run)
	n.claimed.Go("moves", mover.Run)
	n.claimed.Go("conversions", converts.Run)
	n.claimed.Go("subtitles", subtitler.Run)
	n.claimed.Go("tasks", scheduler.Run)
	n.claimed.Go("library scan", found.Run)
	n.claimed.Go("metadata", meta.Run)
	n.claimed.Go("availability", metadata.NewAvailability(metadata.AvailabilityConfig{
		Library: lib.db, Local: local, Targets: goals, Hub: n.hub, Changed: works.ChangedEvent.Bind(n.hub),
		Now: n.today, Released: pipeline.Released, Log: n.rootLog.With("subsystem", "availability"),
	}).Run)
	n.claimed.Go("health", h.health.Run)
	n.claimed.Go("notifications", n.notifications.Run)
	n.claimed.Go("history", n.history.Run)
	n.claimed.Go("first setup", h.firstRun.Run)
	n.env.Test.started(started)
	return nil
}

// openedRoster is the nodes service once it opens: the download clients
// and the Trackers are made before it, since the service reads the
// clients' owner, which reads the Trackers.
type openedRoster struct{ side **nodes.Service }

func (r openedRoster) InStep(node role.NodeID) bool { return (*r.side).InStep(node) }

func (r openedRoster) Standing(node role.NodeID) (nodes.Standing, time.Time) {
	return (*r.side).Standing(node)
}

func (r openedRoster) WrittenOff(name role.Name, node role.NodeID) bool {
	return (*r.side).WrittenOff(name, node)
}

func (r openedRoster) Changed() <-chan struct{} { return (*r.side).Changed() }

func (r openedRoster) Unknowns() []nodes.Unknown { return (*r.side).Unknowns() }

func (r openedRoster) Self() role.NodeID { return (*r.side).Self() }

func (r openedRoster) Serving(name role.Name) []role.NodeID { return (*r.side).Serving(name) }

func (r openedRoster) Holding(name role.Name) []role.NodeID { return (*r.side).Holding(name) }

func (r openedRoster) Holds(name role.Name, node role.NodeID) bool {
	return (*r.side).Holds(name, node)
}

func (r openedRoster) Draining(name role.Name, node role.NodeID) bool {
	return (*r.side).Draining(name, node)
}

func (r openedRoster) Pick(name role.Name, prefer role.NodeID) (role.NodeID, bool) {
	return (*r.side).Pick(name, prefer)
}

func (r openedRoster) Name(node role.NodeID) string { return (*r.side).Name(node) }

// lateCourier reaches the Nodes once the cluster opens: notifications is
// made before the nodes service, which the cluster runs over.
type lateCourier struct{ cluster **rpc.Cluster }

func (c lateCourier) Post(ctx context.Context, node role.NodeID, r notifying.Request) (notifying.Response, error) {
	return notifying.Client{Cluster: *c.cluster}.Post(ctx, node, r)
}

func (c lateCourier) Run(ctx context.Context, node role.NodeID, s notifying.Script) (notifying.Ran, error) {
	return notifying.Client{Cluster: *c.cluster}.Run(ctx, node, s)
}

func (c lateCourier) Send(ctx context.Context, node role.NodeID, s mail.Server, m mail.Message) (mail.Outcome, error) {
	return notifying.Client{Cluster: *c.cluster}.Send(ctx, node, s, m)
}

// lateArtwork is the metadata store once it opens.
type lateArtwork struct{ meta **metadata.Metadata }

func (a lateArtwork) Image(ctx context.Context, ref artwork.Ref, size artwork.Size) (artwork.Image, error) {
	return (*a.meta).Image(ctx, ref, size)
}

func (a lateArtwork) PublicURL(ref artwork.Ref, size artwork.Size) (string, error) {
	return (*a.meta).PublicURL(ref, size)
}

// lateSubtitles is the Controller's Subtitles once it opens: files opens
// before it, since subtitles reads files.
type lateSubtitles struct{ files.Subtitles }

// lateTasks is the Scheduled tasks once they open: the Media Servers and
// grabs are made before them, since the tasks' catalog lists their tasks.
type lateTasks struct{ tasks **tasks.Scheduler }

func (t lateTasks) Every(ctx context.Context, id tasks.TaskID) (*int64, error) {
	return (*t.tasks).Every(ctx, id)
}

func (t lateTasks) EveryIn(ctx context.Context, r storage.Reader[storage.Library], id tasks.TaskID) (*int64, error) {
	return (*t.tasks).EveryIn(ctx, r, id)
}

func (t lateTasks) Next(ctx context.Context, id tasks.TaskID) (time.Time, error) {
	return (*t.tasks).Next(ctx, id)
}

func (t lateTasks) Start(ctx context.Context, st tasks.Start) (tasks.Started, error) {
	return (*t.tasks).Start(ctx, st)
}

func (t lateTasks) SetCadence(ctx context.Context, id tasks.TaskID, v setting.Stored[int64]) error {
	return (*t.tasks).SetCadence(ctx, id, v)
}

// lateSeeder is the Trackers' seeder once they are made: the download
// clients are made before them, since the Trackers read Torrents through
// them.
// lateTypes is the Content Types once the Works open, which open after
// the Download Clients.
type lateTypes struct{ works **works.Works }

func (t lateTypes) Describe(id works.TypeID) (works.Description, bool) {
	return (*t.works).Describe(id)
}

type lateSeeder struct{ trackers **trackers.Trackers }

func (s lateSeeder) Judge(ctx context.Context, r storage.Reader[storage.Library], rows []downloadclients.Row) (map[downloadclients.ID]downloadclients.Judged, error) {
	return (*s.trackers).Seeder().Judge(ctx, r, rows)
}

func (s lateSeeder) EndlessUnits(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) (int64, error) {
	return (*s.trackers).Seeder().EndlessUnits(ctx, r, node)
}

// lateMoves is the Moves once they are made: downloadclients and
// trackers are made first.
type lateMoves struct{ moves **moves.Moves }

func (l lateMoves) Moving(ctx context.Context, r storage.Reader[storage.Library], ids []downloadclients.ID) (map[downloadclients.ID]downloadclients.MovingTo, error) {
	return (*l.moves).Moving(ctx, r, ids)
}

func (l lateMoves) CancelIn(ctx context.Context, tx *storage.Tx[storage.Library], id downloadclients.ID) error {
	return (*l.moves).CancelIn(ctx, tx, id)
}

func (l lateMoves) MovesOf(ctx context.Context, r storage.Reader[storage.Library], ids []downloadclients.ID) (map[downloadclients.ID]downloadclients.Move, error) {
	return (*l.moves).MovesOf(ctx, r, ids)
}

func (l lateMoves) UsesClient(ctx context.Context, r storage.Reader[storage.Library], client downloadclients.ClientID) (int64, error) {
	return (*l.moves).UsesClient(ctx, r, client)
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

// searchingService is the Searching service a half mounts, reaching each
// site through endpoints.searching ahead of the Node's network. Its jobs
// stop with the Node.
func (n *Node) searchingService() *searching.Service {
	client := n.httpClient(n.cfg.Endpoints.Searching.Dial(n.env.Net.DialContext))
	svc := searching.New(searching.Config{
		HTTP: client, Native: port.Driver(n.ports), Timings: n.cfg.Timings.Searching, Log: n.rootLog.With("subsystem", "searching"),
	})
	n.claimed.OnStop("searching jobs", svc.Stop)
	return svc
}

// httpClient is a half's client for the hosts the Node reaches, dialing
// with dial and trusting the OS roots.
func (n *Node) httpClient(dial searching.DialFunc) *http.Client { return httpClient(n.claimed, dial) }

// httpClient is a client that dials with dial and trusts the OS roots,
// whose idle connections close as stack stops.
func httpClient(stack *Stack, dial searching.DialFunc) *http.Client {
	transport := &http.Transport{
		DialContext:         dial,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
	stack.OnStop("http client", func(context.Context) error {
		transport.CloseIdleConnections()
		return nil
	})
	return &http.Client{Transport: transport}
}

// openHolder opens every Content Type's provider half over the Node's one
// call layer, each provider's Line registered once, and the providers'
// tests, in the order status lists them.
func (n *Node) openHolder(client *http.Client) metadata.HolderConfig {
	lines := metadata.NewLines(client, n.userAgent(), n.cfg.Timings.Metadata, n.metadataRand)
	deps := contenttype.ProviderDeps{
		TMDB:  tmdb.New(lines, n.cfg.Endpoints.TMDB, n.cfg.Timings.TMDB),
		TVDB:  tvdb.New(lines, n.cfg.Endpoints.TVDB, n.cfg.Timings.TVDB),
		OMDb:  omdb.New(lines, n.cfg.Endpoints.OMDb, n.cfg.Timings.OMDb),
		Lines: lines,
		Now:   n.today,
	}
	sources := map[works.TypeID]metadata.Source{}
	for _, t := range n.env.Types.All() {
		deps.Log = n.rootLog.With("subsystem", "contenttype."+string(t.ID()))
		sources[t.ID()] = t.OpenProvider(deps, n.cfg.Endpoints.ContentTypes.section(t.ID()))
	}
	return metadata.HolderConfig{Types: sources, Testers: deps.Testers(), Timings: n.cfg.Timings.Metadata, Lines: lines}
}

func (n *Node) userAgent() string {
	return "SlipStream/" + n.env.Version + " ( github.com/jatassi/SlipStream )"
}

// subtitlesHolder is SubtitlesService over the subtitles Lines, whose
// client refuses every redirect, so a key never reaches a host its
// provider didn't name.
func (n *Node) subtitlesHolder(client *http.Client) *subtitling.Holder {
	lines := calls.NewLines(calls.NoRedirects(client), n.userAgent(), n.cfg.Timings.Subtitling.Calls, n.subtitlingRand)
	return subtitling.Serve(subtitleProviders(lines, n.cfg), n.cfg.Timings.Subtitling)
}

// subtitleSpecs is every Subtitle Provider's Spec, for the Controller,
// which calls none of them.
func (n *Node) subtitleSpecs() []subtitling.Spec {
	var out []subtitling.Spec
	for _, p := range subtitleProviders(calls.NewLines(http.DefaultClient, n.userAgent(), n.cfg.Timings.Subtitling.Calls, n.subtitlingRand), n.cfg) {
		out = append(out, p.Spec())
	}
	return out
}

// subtitleProviders is every Subtitle Provider, in its default search
// order.
func subtitleProviders(lines *calls.Lines, cfg Config) []subtitling.Provider {
	return []subtitling.Provider{
		subdl.New(lines, cfg.Endpoints.SubDL, cfg.Timings.SubDL),
		subsource.New(lines, cfg.Endpoints.SubSource, cfg.Timings.SubSource),
		gestdown.New(lines, cfg.Endpoints.Gestdown, cfg.Timings.Gestdown),
	}
}

// openLibraries opens every Content Type's library half, on the
// Controller.
func (n *Node) openLibraries() map[works.TypeID]contenttype.Library {
	libraries := map[works.TypeID]contenttype.Library{}
	for _, t := range n.env.Types.All() {
		libraries[t.ID()] = t.OpenLibrary(contenttype.LibraryDeps{
			Region: metadata.Region,
			Log:    n.rootLog.With("subsystem", "contenttype."+string(t.ID())),
		}, n.cfg.Endpoints.ContentTypes.section(t.ID()))
	}
	return libraries
}

// startWorker starts the Worker half: the holder half over the link to its
// Controller, and call dispatch toward it.
func (n *Node) startWorker(ctx context.Context) error {
	side := n.pairing
	controller := side.Controller()
	front := serving.NewWorker(n.link, controller.Key, n.cfg.Timings.Serving, n.rootLog.With("subsystem", "serving"))
	var holding *role.Holding
	dispatch := rpc.NewDispatch(n.link, n.key.Public(), controller.Key, func(r role.Name) bool { return holding.Holds(r) })
	n.mediaLink = n.newMediaLink()
	rec, _ := side.Record()
	holding, local, copier, err := n.startHolding(ctx, rec.NodeID, role.OverLink(n.link, controller.Key, n.cfg.Timings.Link), dispatch, []role.Held{role.HoldRun(role.Serving, front.Run, front.Release)}, side.Remember)
	if err != nil {
		return err
	}
	searchv1connect.SearchingService.Mount(dispatch, n.searchingService())
	client := n.httpClient(n.env.Net.DialContext)
	metadatav1connect.MetadataService.Mount(dispatch, metadata.NewHolder(n.openHolder(client)))
	subtitlesv1connect.SubtitlesService.Mount(dispatch, n.subtitlesHolder(client))
	mediav1connect.MediaService.Mount(dispatch, media.NewService(n.env.Media, n.downloading, n.store.Holds, n.store.LibraryFile, n.cfg.Timings.Media))
	notifyv1connect.NotifyingService.Mount(dispatch, notifying.Serve(n.cfg.Timings.Mail, n.env.Net.DialContext, n.env.Scripts))
	if err := n.mountSidecars(ctx, dispatch, local); err != nil {
		return err
	}
	storagev1connect.StorageService.Mount(dispatch, n.store)
	storagev1connect.FolderService.Mount(dispatch, n.store)
	downloadingv1connect.DownloadingService.Mount(dispatch, n.downloading)
	mediaserverv1connect.MediaServerLinkService.Mount(dispatch, n.mediaLink.Service())
	convertingv1connect.ConvertingService.Mount(dispatch, converting.Service(n.proofs))
	n.serveSystem(dispatch)
	networkv1connect.NodeNetworkService.Mount(dispatch, netfacts.NewService(n.network))
	housekeepingv1connect.HousekeepingService.Mount(dispatch, housekeeping.Service{Local: local})
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{self: rec.NodeID, peers: side, dispatch: dispatch, holding: holding, serving: front, copier: copier})
	n.link.Keep(controller)
	n.claimed.Go("pairing", side.Run)
	n.claimed.Go("superseded", func(ctx context.Context) error { return side.Tell(ctx, n.cfg.Timings.Pairing.SupersededDial) })
	n.claimed.Go("serving", front.Follow)
	n.env.Test.started(Half{Link: n.link, Dispatch: dispatch, Local: local})
	return nil
}

// removed answers the Controller's removal of this Worker: the holder
// half forgets its cached intents, so its Roles stop and a later Join
// can't resume them, waits a bounded time for its Node-wide channels to
// let go, such as the network channel deleting its router mappings, and
// the Node restarts Unclaimed.
func (n *Node) removed(ctx context.Context) {
	if h := n.half.Load(); h != nil {
		ctx, cancel := context.WithTimeout(ctx, n.cfg.Timings.Role.Release)
		defer cancel()
		if err := h.holding.Reset(ctx); err != nil {
			n.log.Error("forget the removed Worker's Roles", "err", err.Error())
		}
	}
	if err := n.Restart(); err != nil {
		n.log.Error("restart the removed Worker", "err", err.Error())
	}
}

func (n *Node) mountSidecars(ctx context.Context, dispatch *rpc.Dispatch, local *storage.DB[storage.Local]) error {
	sidecars, err := storing.NewSidecars(ctx, n.store, local, n.env.DataDir.SpeechMapsDir(), n.env.Speech, n.cfg.Timings.Sidecars)
	if err != nil {
		return err
	}
	storagev1connect.SidecarService.Mount(dispatch, sidecars)
	return nil
}

// startHolding opens local.db and runs the holder half toward upstream,
// holding the half's own Roles beside every Node's, and the Node's
// Transfers engine over local.db. Its Roles call the Controller through
// dispatch. A Worker's holder hands superseded each Controller key its
// node-scope intent names as replaced. It builds the Node's network facts
// over local.db, which keeps self's router mappings.
func (n *Node) startHolding(ctx context.Context, self role.NodeID, upstream role.Upstream, dispatch *rpc.Dispatch, own []role.Held, superseded func([]link.SupersededProof)) (*role.Holding, *storage.DB[storage.Local], *transfers.Node, error) {
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), n.cfg.Timings.Storage, n.rootLog.With("subsystem", "storage"))
	if err != nil {
		return nil, nil, nil, err
	}
	n.claimed.OnStop("local.db", func(context.Context) error { return local.Close() })
	n.network = n.newNetwork(self, local)
	n.claimed.OnStop("torrent resume data", n.downloads.Shutdown(local))
	copier := transfers.NewNode(transfers.NodeConfig{
		Local:     local,
		Disk:      n.disk,
		Link:      n.link,
		Sources:   opener{store: n.store, torrents: n.downloading},
		Roots:     n.store,
		Downloads: n.downloading,
		Scratch:   n.scratch,
		Timings:   n.cfg.Timings.Transfers,
		Retry:     n.cfg.Timings.Backups.WindowsRetry,
		Hooks:     n.env.Test.Transfers,
		Log:       n.rootLog.With("subsystem", "transfers"),
	})
	holding, err := role.NewHolding(ctx, role.HoldingConfig{
		Local:      local,
		Upstream:   upstream,
		Held:       slices.Concat(n.heldRoles(local, dispatch, copier), own, n.env.Test.held(local)),
		Superseded: superseded,
		Timings:    n.cfg.Timings.Role,
		Rand:       n.roleRand,
		Log:        n.rootLog.With("subsystem", "role"),
	})
	if err != nil {
		return nil, nil, nil, err
	}
	n.claimed.Go("role", holding.Run)
	return holding, local, copier, nil
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

// housekeepingSteps is Housekeeping's prunes in their units' order. The
// Grabs step runs before the Imports step, so the Imports step sees the
// Grabs it deleted.
func housekeepingSteps(registry *indexers.Indexers, servers *mediaservers.MediaServers, h *history.Service, pipeline *grabs.Grabs, found *imports.Imports, converts *conversions.Conversions, shelf *files.Files, subtitler *subtitles.Subtitles) []housekeeping.Step {
	return []housekeeping.Step{
		housekeeping.Plain("indexer-queries", "Indexer queries and sessions", tasks.Runs, registry.Prune), servers.Housekeeping(),
		h.Retention(), pipeline.Retention(), found.Retention(), converts.Retention(), shelf.Retention(), subtitler.Retention(),
	}
}

// owners is each Role's owner: the feature that keeps its work.
func owners(clients *downloadclients.Clients, servers *mediaservers.MediaServers, notes *notifications.Service) []nodes.Owner {
	return []nodes.Owner{
		indexers.SearchingOwner,
		clients.DownloadingOwner(),
		roots.StorageOwner(files.RootContents{}),
		conversions.Owner,
		subtitles.Owner,
		metadata.Owner,
		servers.Owner(),
		notes.Owner(),
		nodes.Unowned(role.Relaying),
		nodes.Unowned(role.Serving),
	}
}

// otherCopies is the bytes other owners expect on node's volume: the
// copies S48 (Moves) granted and the outputs unfinished Conversions write.
// The Conversions are made after imports, which reads them.
func otherCopies(converts **conversions.Conversions) imports.OtherCopies {
	return func(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID, volume string) (int64, error) {
		moved, err := moves.OtherCopies(ctx, r, node, volume)
		if err != nil {
			return 0, err
		}
		made, err := (*converts).OutputsIn(ctx, r, node, volume)
		return moved + made, err
	}
}

// writing is which of fs a Move waits for: a Speech Map or Sidecar
// written for it, or an unfinished Conversion reading or making it.
func writing(ctx context.Context, r storage.Reader[storage.Library], fs []files.ID) (map[files.ID]bool, error) {
	out := map[files.ID]bool{}
	for _, of := range []func(context.Context, storage.Reader[storage.Library], []files.ID) (map[files.ID]bool, error){subtitles.Writing, conversions.Open} {
		busy, err := of(ctx, r, fs)
		if err != nil {
			return nil, err
		}
		for id, b := range busy {
			out[id] = out[id] || b
		}
	}
	return out, nil
}

// scratchVolume is the id of the volume that holds node's scratch folder.
func scratchVolume(role.NodeID) (string, bool) { return "", false }

// heldRoles is every Role this binary runs, and the Node-wide channels
// every Node holds, one held line each.
func (n *Node) heldRoles(local *storage.DB[storage.Local], dispatch *rpc.Dispatch, copier *transfers.Node) []role.Held {
	convert := converting.Hold(converting.HolderConfig{
		Converter: n.converter, Proofs: n.proofs, Media: n.env.Media, Scratch: n.scratch, Override: n.ffmpegPath,
		Local: n.store.Local, Disk: n.disk, DB: local,
		Timings: n.cfg.Timings.Converting, Log: n.rootLog.With("subsystem", "converting"),
	})
	return []role.Held{
		downloading.Def.Hold(n.downloading.Run(local, downloadingv1connect.DownloadingControlService.Call(dispatch)), n.downloading.Release),
		storing.Def.Hold(n.store.Run, n.store.Release),
		transfers.Def.Hold(copier.Run, copier.Release),
		mediaserverlink.Def.Hold(n.mediaLink.Run(local, mediaserverv1connect.MediaServerControlService.Call(dispatch)), n.mediaLink.Release),
		netmodel.Def.Hold(n.network.Watch, n.network.Release),
		converting.Def.Hold(convert.Run, convert.Release),
	}
}

// newMediaLink is the half's Media Server link holder.
func (n *Node) newMediaLink() *mediaserverlink.Link {
	return mediaserverlink.NewLink(mediaserverlink.Config{
		Kinds:   mediaserverlink.Kinds,
		HTTP:    n.httpClient(n.env.Net.DialContext),
		Timings: n.cfg.Timings.MediaServers,
		Log:     n.rootLog.With("subsystem", "media server link"),
	})
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

func (p claimPeers) Told(from link.Key, s link.Superseded) {
	if h := p.n.half.Load(); h != nil {
		h.peers.Told(from, s)
	}
}

// Unknown is link.Config.Unknown: a Controller lists each Node it
// doesn't know that says Hello, for the Reconciling page.
func (p claimPeers) Unknown(key link.Key, peer link.NodeInfo, from netip.AddrPort) {
	if h := p.n.half.Load(); h != nil && h.nodes != nil {
		h.nodes.Unknown(key, peer, from)
	}
}

// claimStreams serves Serve streams from a Node that serves on the
// Controller, and resets them on any other Node.
type claimStreams struct{ n *Node }

func (c claimStreams) serve(peer link.Key, s link.Stream) {
	if h := c.n.half.Load(); h != nil && h.nodes != nil {
		if id, ok := h.nodes.NodeOf(peer); ok && h.nodes.Serves(role.Serving, id) {
			c.n.served.Offer(id, s)
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

// claimLanes serves lanes, and route 3's Transfer streams, with the
// half's Transfers engine, and refuses them while the Node is Unclaimed.
type claimLanes struct{ n *Node }

func (c claimLanes) copier() *transfers.Node {
	if h := c.n.half.Load(); h != nil {
		return h.copier
	}
	return nil
}

func (c claimLanes) Admit(key link.Key) bool {
	t := c.copier()
	return t != nil && t.Admit(key)
}

func (c claimLanes) Serve(ctx context.Context, key link.Key, conn net.Conn) {
	if t := c.copier(); t != nil {
		t.Serve(ctx, key, conn)
	}
}

func (c claimLanes) stream(key link.Key, s link.Stream) {
	if t := c.copier(); t != nil {
		t.ServeStream(key, s)
		return
	}
	_ = s.Reset()
}

func notAController() *linkv1.PairResponse {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER}}}
}

func metainfoOn(ctx context.Context, cluster *rpc.Cluster, node role.NodeID, hash bencode.InfoHash, client *downloadclients.ClientID) ([]byte, error) {
	req := &dlv1.MetainfoRequest{InfoHash: hash[:]}
	if client != nil {
		req.ClientId = client.Bytes()
	}
	res, err := downloadingv1connect.DownloadingService.On(cluster, node).Metainfo(ctx, req)
	return res.GetMetainfo(), err
}

func lookOn(ctx context.Context, cluster *rpc.Cluster, node role.NodeID, folder string, looked []downloadclients.LookFile) ([]bool, error) {
	req := &dlv1.LookRequest{Folder: folder, Entries: make([]*dlv1.LookEntry, len(looked))}
	for i, f := range looked {
		req.Entries[i] = &dlv1.LookEntry{Name: f.Name, Size: f.Size}
	}
	res, err := downloadingv1connect.DownloadingService.On(cluster, node).Look(ctx, req)
	return res.GetFound(), err
}

// opener serves a Transfer's sources on this Node: a library file from
// the root that holds it, read even while the Node drains its Storage,
// and every other source from its Download Client.
type opener struct {
	store    *storing.Store
	torrents transfers.Opener
}

func (o opener) Open(ref source.Ref) (string, error) {
	lib, ok := ref.(source.Library)
	if !ok {
		return o.torrents.Open(ref)
	}
	rel, err := storing.ParseRel(lib.Rel)
	if err != nil {
		return "", &source.Absent{Reason: source.Gone, Detail: err.Error()}
	}
	root, err := storage.UUIDFromBytes(lib.Root[:])
	if err != nil {
		return "", err
	}
	return o.store.OpenLibrary(context.Background(), root, rel)
}

// rootBusy is what uses root: the open Moves naming it, the unfinished
// Imports into it and the live Transfers into or out of it.
func rootBusy(ctx context.Context, r storage.Reader[storage.Library], root roots.ID) (roots.Busy, error) {
	movesIn, err := moves.Busy(ctx, r, root)
	if err != nil {
		return roots.Busy{}, err
	}
	importsIn, err := imports.RootImports{}.Into(ctx, r, root, nil)
	if err != nil {
		return roots.Busy{}, err
	}
	transfersIn, err := transfers.UsingRoot(ctx, r, root)
	if err != nil {
		return roots.Busy{}, err
	}
	return roots.Busy{Moves: int(movesIn), Imports: int(importsIn), Transfers: transfersIn}, nil
}
