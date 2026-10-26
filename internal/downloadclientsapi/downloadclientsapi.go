// Package downloadclientsapi is the built-in torrent clients' operations:
// the Torrents they hold and the Admin's actions on them, each downloading
// Node's client card and settings, and the thresholds for a bad release.
package downloadclientsapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/settingapi"
	"github.com/jatassi/slipstream/internal/storage"
)

// Clients is the Node's download clients owner and its roster.
type Clients interface {
	// DownloadClients is the record of every Node's built-in client,
	// false unless the Node is a Controller.
	DownloadClients() (*downloadclients.Clients, bool)
	// Nodes is the nodes service, false unless the Node is a Controller.
	Nodes() (*nodes.Service, bool)
}

func open(c Clients) (*downloadclients.Clients, *nodes.Service) {
	clients, ok := c.DownloadClients()
	roster, known := c.Nodes()
	if !ok || !known {
		panic("downloadclientsapi: an Admin operation ran on a Node that is no Controller")
	}
	return clients, roster
}

var nodeNotFound = api.NewCode(http.StatusNotFound, "node_not_found", "No Node has this id")

type settings struct {
	DownloadLimit   setting.RateField     `json:"downloadLimit"`
	UploadLimit     setting.RateField     `json:"uploadLimit"`
	ActiveDownloads setting.CountField    `json:"activeDownloads"`
	ActiveChecks    setting.CountField    `json:"activeChecks"`
	Connections     setting.CountField    `json:"connections"`
	UTP             setting.SwitchField   `json:"utp"`
	StalledAfter    setting.DurationField `json:"stalledAfter"`
}

type settingsPatch struct {
	DownloadLimit   api.Patch[int64] `json:"downloadLimit,omitempty" doc:"Bytes a second, 0 for no limit; null resets it to its default"`
	UploadLimit     api.Patch[int64] `json:"uploadLimit,omitempty" doc:"Bytes a second, 0 for no limit; null resets it to its default"`
	ActiveDownloads api.Patch[int64] `json:"activeDownloads,omitempty" doc:"0 for no limit; null resets it to its default"`
	ActiveChecks    api.Patch[int64] `json:"activeChecks,omitempty" doc:"null resets it to its default"`
	Connections     api.Patch[int64] `json:"connections,omitempty" doc:"null resets it to its default"`
	UTP             api.Patch[bool]  `json:"utp,omitempty" doc:"null resets it to its default"`
	StalledAfter    api.Patch[int64] `json:"stalledAfter,omitempty" doc:"Seconds; null resets it to its default"`
}

type nodeInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
}

type settingsInput struct {
	NodeID string `path:"nodeId" format:"uuid"`
	Body   settingsPatch
}

type settingsOutput struct{ Body settings }

func settingsOf(o downloadclients.NodeOverrides) *settingsOutput {
	return &settingsOutput{Body: settings{
		DownloadLimit:   downloadclients.DownloadLimit.Field(o.DownloadLimit),
		UploadLimit:     downloadclients.UploadLimit.Field(o.UploadLimit),
		ActiveDownloads: downloadclients.ActiveDownloads.Field(o.ActiveDownloads),
		ActiveChecks:    downloadclients.ActiveChecks.Field(o.ActiveChecks),
		Connections:     downloadclients.Connections.Field(o.Connections),
		UTP:             downloadclients.UTP.Field(o.UTP),
		StalledAfter:    downloadclients.StalledAfter.Field(o.StalledAfter),
	}}
}

func change(b settingsPatch) (downloadclients.NodeChange, error) {
	var c downloadclients.NodeChange
	for _, f := range []struct {
		patch api.Patch[int64]
		parse func(*int64) (setting.Stored[int64], error)
		into  **setting.Stored[int64]
	}{
		{b.DownloadLimit, downloadclients.DownloadLimit.Parse, &c.DownloadLimit},
		{b.UploadLimit, downloadclients.UploadLimit.Parse, &c.UploadLimit},
		{b.ActiveDownloads, downloadclients.ActiveDownloads.Parse, &c.ActiveDownloads},
		{b.ActiveChecks, downloadclients.ActiveChecks.Parse, &c.ActiveChecks},
		{b.Connections, downloadclients.Connections.Parse, &c.Connections},
		{b.StalledAfter, downloadclients.StalledAfter.Parse, &c.StalledAfter},
	} {
		if !f.patch.Sent() {
			continue
		}
		v, err := f.parse(f.patch.Value())
		if err != nil {
			return c, settingapi.Refused(err)
		}
		*f.into = &v
	}
	if b.UTP.Sent() {
		c.UTP = new(downloadclients.UTP.Parse(b.UTP.Value()))
	}
	return c, nil
}

// node is the Node id names, nodeNotFound unless the roster has it.
func node(ctx context.Context, roster *nodes.Service, id string) (role.NodeID, error) {
	u, err := storage.ParseUUID(id)
	if err != nil {
		return role.NodeID{}, nodeNotFound
	}
	n := nodes.ID{UUID: u}
	if _, err := roster.Get(ctx, n); errors.Is(err, nodes.ErrNotFound) {
		return role.NodeID{}, nodeNotFound
	} else if err != nil {
		return role.NodeID{}, err
	}
	return n, nil
}

type handlers struct{ clients Clients }

func (h handlers) getSettings(ctx context.Context, _ identity.Admin, in *nodeInput) (*settingsOutput, error) {
	clients, roster := open(h.clients)
	id, err := node(ctx, roster, in.NodeID)
	if err != nil {
		return nil, err
	}
	o, err := clients.NodeSettings(ctx, id)
	if err != nil {
		return nil, err
	}
	return settingsOf(o), nil
}

func (h handlers) setSettings(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
	clients, roster := open(h.clients)
	id, err := node(ctx, roster, in.NodeID)
	if err != nil {
		return nil, err
	}
	c, err := change(in.Body)
	if err != nil {
		return nil, err
	}
	o, err := clients.SetNodeSettings(ctx, id, c)
	if err != nil {
		return nil, err
	}
	return settingsOf(o), nil
}

type clientsList struct {
	Cards      []clientCard `json:"cards" nullable:"false" doc:"Each Node assigned Downloading or draining it, in Node order"`
	Thresholds thresholds   `json:"thresholds"`
}

type clientCard struct {
	NodeID    string           `json:"nodeId" format:"uuid"`
	NodeName  string           `json:"nodeName"`
	InStep    bool             `json:"inStep" doc:"The Node is in step; while it isn't the card has no client"`
	Client    *clientFacts     `json:"client,omitempty" doc:"The built-in client as it reports itself, absent until it does and while the Node is out of step"`
	Limits    limits           `json:"limits"`
	Totals    totals           `json:"totals" doc:"Added up over the Node's live Torrents"`
	Torrents  int              `json:"torrents" minimum:"0" doc:"The Node's live Torrents"`
	Downloads *downloadsVolume `json:"downloads,omitempty" doc:"The downloads folder's volume as last sampled, absent until the first sample"`
	DiskGuard *diskGuard       `json:"diskGuard,omitempty" doc:"Set while the disk guard holds the Node's new downloads: its downloads volume is below its error level"`
}

type downloadsVolume struct {
	Mount     string    `json:"mount" doc:"As the Node names it, the same as Storage names the volume"`
	Name      string    `json:"name" doc:"Such as D: (Media)"`
	Total     int64     `json:"total" minimum:"0" doc:"Bytes"`
	Free      int64     `json:"free" minimum:"0" doc:"Bytes"`
	SampledAt time.Time `json:"sampledAt"`
}

type diskGuard struct {
	Sentence   string `json:"sentence" doc:"The guard's sentence, as Health shows it"`
	Hint       string `json:"hint" doc:"What explains the sentence: the volume, its free space, its error level and what to do"`
	ErrorBelow int64  `json:"errorBelow" minimum:"0" doc:"The volume's error level, in bytes"`
}

type clientFacts struct {
	Libtorrent   string     `json:"libtorrent" doc:"libtorrent's version"`
	PeerIDPrefix string     `json:"peerIdPrefix" doc:"The peer ID prefix it announces, which a tracker's allow list names"`
	UserAgent    string     `json:"userAgent" doc:"The user agent it announces"`
	Listen       string     `json:"listen" doc:"The listen address and port it is set to"`
	Bound        string     `json:"bound,omitempty" doc:"The address and port it listens on, absent while it has no listener"`
	ListenError  string     `json:"listenError,omitempty" doc:"Why it has no listener"`
	Reach        string     `json:"reach" enum:"reachable,unproven" doc:"reachable once a peer from the internet has connected to the address it listens on now"`
	ReachedAt    *time.Time `json:"reachedAt,omitempty" doc:"When the first such peer connected"`
}

type limits struct {
	DownloadLimit   int64 `json:"downloadLimit" minimum:"0" doc:"Bytes a second, 0 for no limit"`
	UploadLimit     int64 `json:"uploadLimit" minimum:"0" doc:"Bytes a second, 0 for no limit"`
	ActiveDownloads int64 `json:"activeDownloads" minimum:"0" doc:"0 for no limit"`
	ActiveChecks    int64 `json:"activeChecks" minimum:"1"`
	Connections     int64 `json:"connections" minimum:"1"`
	UTP             bool  `json:"utp"`
	StalledAfter    int64 `json:"stalledAfterSeconds" minimum:"1"`
}

type totals struct {
	Uploaded   int64 `json:"uploaded" minimum:"0" doc:"Bytes"`
	Downloaded int64 `json:"downloaded" minimum:"0" doc:"Bytes"`
	Seeding    int64 `json:"seedingSeconds" minimum:"0"`
	Active     int64 `json:"activeSeconds" minimum:"0"`
}

type clientsListOutput struct{ Body clientsList }

type thresholds struct {
	FaultStall      setting.DurationField `json:"faultStall"`
	UnregisteredGap setting.DurationField `json:"unregisteredGap"`
}

type thresholdsPatch struct {
	FaultStall      api.Patch[int64] `json:"faultStall,omitempty" doc:"Seconds; null resets it to its default"`
	UnregisteredGap api.Patch[int64] `json:"unregisteredGap,omitempty" doc:"Seconds; null resets it to its default"`
}

type thresholdsInput struct{ Body thresholdsPatch }

type thresholdsOutput struct{ Body thresholds }

func thresholdsOf(o downloadclients.ThresholdOverrides) thresholds {
	return thresholds{FaultStall: downloadclients.FaultStall.Field(o.FaultStall), UnregisteredGap: downloadclients.UnregisteredGap.Field(o.UnregisteredGap)}
}

func cardOf(c downloadclients.Card, name string) clientCard {
	s := c.Settings
	out := clientCard{
		NodeID: c.Node.String(), NodeName: name, InStep: c.InStep, Torrents: c.Torrents,
		Limits: limits{
			DownloadLimit: s.DownloadLimit, UploadLimit: s.UploadLimit, ActiveDownloads: s.ActiveDownloads, ActiveChecks: s.ActiveChecks,
			Connections: s.Connections, UTP: s.UTP, StalledAfter: int64(s.StalledAfter / time.Second),
		},
		Totals: totals{
			Uploaded: c.Totals.Uploaded, Downloaded: c.Totals.Downloaded,
			Seeding: int64(c.Totals.Seeding / time.Second), Active: int64(c.Totals.Active / time.Second),
		},
	}
	if g := c.Guard; g != nil {
		sentence, hint := g.Words(name)
		out.DiskGuard = &diskGuard{Sentence: sentence, Hint: hint, ErrorBelow: g.ErrorBelow.Bytes}
	}
	if v := c.Volume; v != nil {
		out.Downloads = &downloadsVolume{Mount: v.Mount, Name: v.Name, Total: v.Total, Free: v.Free, SampledAt: v.SampledAt}
	}
	if f := c.Facts; f != nil {
		out.Client = &clientFacts{
			Libtorrent: f.Libtorrent, PeerIDPrefix: f.PeerIDPrefix, UserAgent: f.UserAgent, Listen: f.Listen, Bound: f.Bound,
			ListenError: f.ListenError, Reach: string(f.Reach()), ReachedAt: when(f.ReachedAt),
		}
	}
	return out
}

func (h handlers) listClients(ctx context.Context, _ identity.Admin, _ *struct{}) (*clientsListOutput, error) {
	clients, roster := open(h.clients)
	cards, err := clients.Cards(ctx)
	if err != nil {
		return nil, err
	}
	names, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	o, err := clients.Thresholds(ctx)
	if err != nil {
		return nil, err
	}
	out := clientsList{Cards: make([]clientCard, len(cards)), Thresholds: thresholdsOf(o)}
	for i, c := range cards {
		out.Cards[i] = cardOf(c, names[c.Node].Name)
	}
	return &clientsListOutput{Body: out}, nil
}

func (h handlers) setThresholds(ctx context.Context, _ identity.Admin, in *thresholdsInput) (*thresholdsOutput, error) {
	clients, _ := open(h.clients)
	var ch downloadclients.ThresholdChange
	for _, f := range []struct {
		patch api.Patch[int64]
		parse func(*int64) (setting.Stored[int64], error)
		into  **setting.Stored[int64]
	}{
		{in.Body.FaultStall, downloadclients.FaultStall.Parse, &ch.FaultStall},
		{in.Body.UnregisteredGap, downloadclients.UnregisteredGap.Parse, &ch.UnregisteredGap},
	} {
		if !f.patch.Sent() {
			continue
		}
		v, err := f.parse(f.patch.Value())
		if err != nil {
			return nil, settingapi.Refused(err)
		}
		*f.into = &v
	}
	o, err := clients.SetThresholds(ctx, ch)
	if err != nil {
		return nil, err
	}
	return &thresholdsOutput{Body: thresholdsOf(o)}, nil
}

func Routes(r *api.Router, clients Clients) {
	o := r.Owner("downloadclients", api.Admin)
	h := handlers{clients}

	api.Register(o, api.Admin, api.Op{
		ID:      "listTorrents",
		Method:  http.MethodGet,
		Path:    "/torrents",
		Summary: "List the Torrents every Node's built-in client holds, with their state and counts",
		Status:  http.StatusOK,
	}, h.listTorrents)

	api.Register(o, api.Admin, api.Op{
		ID:      "getTorrent",
		Method:  http.MethodGet,
		Path:    "/torrents/{id}",
		Summary: "Get a Torrent with its trackers, files and Grabs, live from its Node while the Node responds",
		Status:  http.StatusOK,
		Errors:  []*api.Code{torrentNotFound},
	}, h.getTorrent)

	api.Register(o, api.Admin, api.Op{
		ID:      "actOnTorrents",
		Method:  http.MethodPost,
		Path:    "/torrents/actions",
		Summary: "Pause, resume, announce, recheck, mark failed or remove Torrents, and return the result for each",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noPreview},
	}, h.actOnTorrents)

	api.Register(o, api.Admin, api.Op{
		ID:      "listDownloadClients",
		Method:  http.MethodGet,
		Path:    "/download-clients",
		Summary: "List each downloading Node's built-in client, and the thresholds for a bad release",
		Status:  http.StatusOK,
	}, h.listClients)

	api.Register(o, api.Admin, api.Op{
		ID:      "setDownloadClientThresholds",
		Method:  http.MethodPatch,
		Path:    "/download-clients/thresholds",
		Summary: "Change when a release counts as bad, which every downloading Node applies at once",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, h.setThresholds)

	api.Register(o, api.Admin, api.Op{
		ID:      "getDownloadClientSettings",
		Method:  http.MethodGet,
		Path:    "/download-clients/{nodeId}/settings",
		Summary: "Get a Node's built-in torrent client settings",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound},
	}, h.getSettings)

	api.Register(o, api.Admin, api.Op{
		ID:      "setDownloadClientSettings",
		Method:  http.MethodPatch,
		Path:    "/download-clients/{nodeId}/settings",
		Summary: "Change a Node's built-in torrent client settings, which the Node applies at once",
		Status:  http.StatusOK,
		Errors:  []*api.Code{nodeNotFound, settingapi.ErrOutOfRange},
	}, h.setSettings)
}
