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
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

var (
	clientTestFailed   = api.NewCarrying[[]string](http.StatusUnprocessableEntity, "download_client_test_failed", "The Download Client's Test failed", "fields")
	clientChooseDaemon = api.NewCarrying[[]daemon](http.StatusUnprocessableEntity, "download_client_choose_daemon", "Choose the daemon SlipStream uses", "daemons")
	clientUnavailable  = api.NewCode(http.StatusBadGateway, "download_client_unavailable", "The Test couldn't run")
	clientInUse        = api.NewCode(http.StatusConflict, "download_client_in_use", "New Grabs go to this client")
	clientHasTorrents  = api.NewCode(http.StatusConflict, "download_client_has_torrents", "This client holds torrents SlipStream tracks")
	clientAddressTaken = api.NewCode(http.StatusConflict, "download_client_address_taken", "Another Download Client already uses this address")
	clientNameTaken    = api.NewCode(http.StatusConflict, "download_client_name_taken", "Another Download Client has that name")
	clientNotFound     = api.NewCode(http.StatusNotFound, "download_client_not_found", "No Download Client has this id")
	clientFieldRefused = api.NewCode(http.StatusUnprocessableEntity, "download_client_field_refused", "SlipStream can't use a value in this client's settings")
	nodeNotDownloading = api.NewCode(http.StatusConflict, "node_not_downloading", "That Node doesn't hold Downloading")
	clientImporting    = api.NewCode(http.StatusConflict, "download_client_importing", "A Grab is importing from this client")
)

// answer is err as the problem an Admin reads, naming clients and Nodes
// by names.
func answer(err error, names map[nodes.ID]nodes.Named, kind clients.Kind) error {
	var (
		failed  *downloadclients.TestFailedError
		away    *downloadclients.UnavailableError
		inUse   *downloadclients.InUseError
		held    *downloadclients.HasTorrentsError
		imports *downloadclients.ImportingError
		taken   *downloadclients.AddressTakenError
		refused *downloadclients.RefusedError
	)
	name := func(n role.NodeID) string { return names[n].Name }
	switch {
	case errors.As(err, &failed):
		says := downloadclients.Says(failed.Problem, failed.Subject, name(failed.Node))
		if c := failed.Problem.Cause; c == clients.DaemonNeeded || c == clients.WrongDaemon {
			return clientChooseDaemon.Explained(says, daemonsOf(failed.Daemons))
		}
		return clientTestFailed.Explained(says, fieldsOf(failed.Problem.Cause, failed.Subject.Kind))
	case errors.As(err, &away):
		return clientUnavailable.Because(downloadclients.Unreached(name(away.Node)))
	case errors.As(err, &inUse):
		return clientInUse.Because(downloadclients.InUse(name(inUse.Node)))
	case errors.As(err, &held):
		return clientHasTorrents.Because(downloadclients.HasTorrents(held.Name, held.Count, held.Moving))
	case errors.As(err, &imports):
		return clientImporting.Because(downloadclients.Importing(imports.Name))
	case errors.As(err, &taken):
		return clientAddressTaken.Because(downloadclients.AddressTaken(taken.Name, name(taken.Node)))
	case errors.As(err, &refused):
		return clientFieldRefused.On(string(refused.Field), downloadclients.Refused(refused, kind))
	case errors.Is(err, downloadclients.ErrNameTaken):
		return clientNameTaken
	case errors.Is(err, downloadclients.ErrNoClient):
		return clientNotFound
	case errors.Is(err, downloadclients.ErrNotDownloading):
		return nodeNotDownloading
	}
	return settingapi.Refused(err)
}

// fieldsOf is the fields c names that kind takes.
func fieldsOf(c clients.Cause, kind clients.Kind) []string {
	out := []string{}
	for _, f := range c.Fields() {
		if f != clients.FieldUsername || kind.Facts().SignIn.Username != clients.NotTaken {
			out = append(out, string(f))
		}
	}
	return out
}

func daemonsOf(ds []clients.Daemon) []daemon {
	out := []daemon{}
	for _, d := range ds {
		out = append(out, daemon{Host: d.Host, Port: d.Port})
	}
	return out
}

type clientFolder struct {
	Client string `json:"client" minLength:"1" maxLength:"4096" doc:"The folder as the client reports it, such as /downloads"`
	Node   string `json:"node" minLength:"1" maxLength:"4096" doc:"The same folder on the client's Node, such as D:\\Torrents"`
}

type daemon struct {
	Host string `json:"host" minLength:"1" maxLength:"253"`
	Port int    `json:"port" minimum:"1" maximum:"65535"`
}

type newClient struct {
	NodeID        string         `json:"nodeId" format:"uuid" doc:"A Node that holds Downloading"`
	Kind          string         `json:"kind" enum:"qbittorrent,transmission,deluge,rtorrent"`
	Name          string         `json:"name,omitempty" maxLength:"100" doc:"Absent for the kind's name, such as Transmission"`
	Address       string         `json:"address" minLength:"1" maxLength:"2048" doc:"The client's whole address, such as http://localhost:9091/transmission/rpc; with no path it takes the kind's usual one"`
	Username      string         `json:"username,omitempty" maxLength:"256"`
	Password      string         `json:"password,omitempty" maxLength:"1024" doc:"Write-only"`
	APIKey        string         `json:"apiKey,omitempty" maxLength:"1024" doc:"qBittorrent 5.2 and later. Write-only"`
	Label         string         `json:"label,omitempty" maxLength:"100" doc:"Absent for slipstream"`
	Folder        string         `json:"folder,omitempty" maxLength:"4096" doc:"The download folder as the client sees it; absent for the client's default"`
	ClientFolders []clientFolder `json:"clientFolders,omitempty" maxItems:"50"`
	Daemon        *daemon        `json:"daemon,omitempty" doc:"Deluge's daemon; absent for the only one"`
	CheckEvery    *int64         `json:"checkEvery,omitempty" doc:"Seconds; absent for the default"`
	ImportDelay   *int64         `json:"importDelay,omitempty" doc:"Seconds; absent for the default"`
	On            *bool          `json:"on,omitempty" doc:"Absent for on"`
}

func nodeOf(raw string) role.NodeID {
	u, _ := storage.ParseUUID(raw)
	return role.NodeID{UUID: u}
}

func (n newClient) draft() (downloadclients.Draft, error) {
	kind := clients.Kind(n.Kind)
	d := downloadclients.Draft{
		Node: nodeOf(n.NodeID), Kind: kind, Name: n.Name, Username: n.Username,
		Secrets: clients.Secrets{Password: n.Password, APIKey: n.APIKey}, Folder: n.Folder, On: n.On == nil || *n.On,
	}
	var err error
	if d.Address, err = address(n.Address, kind); err != nil {
		return d, err
	}
	raw := n.Label
	if raw == "" {
		raw = clients.DefaultLabel
	}
	if d.Label, err = label(raw, kind); err != nil {
		return d, err
	}
	if d.Folders, err = folders(n.ClientFolders); err != nil {
		return d, err
	}
	if n.Daemon != nil {
		d.Daemon = &clients.Daemon{Host: n.Daemon.Host, Port: n.Daemon.Port}
	}
	if d.CheckEvery, err = downloadclients.CheckEvery.Parse(n.CheckEvery); err != nil {
		return d, err
	}
	d.ImportDelay, err = downloadclients.ImportDelay.Parse(n.ImportDelay)
	return d, err
}

func address(raw string, kind clients.Kind) (clients.Address, error) {
	a, err := clients.ParseAddress(raw, kind)
	if err != nil {
		return a, &downloadclients.RefusedError{Field: clients.FieldAddress, Err: err}
	}
	return a, nil
}

func label(raw string, kind clients.Kind) (clients.Label, error) {
	l, err := clients.ParseLabel(raw, kind)
	if err != nil {
		return l, &downloadclients.RefusedError{Field: clients.FieldLabel, Err: err}
	}
	return l, nil
}

func folders(in []clientFolder) (clients.Folders, error) {
	pairs := make([]clients.Folder, len(in))
	for i, f := range in {
		pairs[i] = clients.Folder{Client: f.Client, Node: f.Node}
	}
	f, err := clients.ParseFolders(pairs)
	if err != nil {
		return f, &downloadclients.RefusedError{Field: clients.FieldFolders, Err: err}
	}
	return f, nil
}

type clientEdit struct {
	NodeID        *string           `json:"nodeId,omitempty" format:"uuid" doc:"Another Node that holds Downloading; a client with Torrents can't move"`
	Kind          *string           `json:"kind,omitempty" enum:"qbittorrent,transmission,deluge,rtorrent"`
	Name          *string           `json:"name,omitempty" maxLength:"100" doc:"Empty for the kind's name"`
	Address       *string           `json:"address,omitempty" minLength:"1" maxLength:"2048"`
	Username      *string           `json:"username,omitempty" maxLength:"256" doc:"Empty signs in with none"`
	Password      string            `json:"password,omitempty" maxLength:"1024" doc:"Write-only; empty keeps the saved one"`
	APIKey        string            `json:"apiKey,omitempty" maxLength:"1024" doc:"Write-only; empty keeps the saved one"`
	Label         *string           `json:"label,omitempty" minLength:"1" maxLength:"100"`
	Folder        *string           `json:"folder,omitempty" maxLength:"4096" doc:"Empty for the client's default"`
	ClientFolders *[]clientFolder   `json:"clientFolders,omitempty" maxItems:"50"`
	Daemon        api.Patch[daemon] `json:"daemon,omitempty" doc:"Null for Deluge's only daemon"`
	CheckEvery    api.Patch[int64]  `json:"checkEvery,omitempty" doc:"Seconds; null resets it to its default"`
	ImportDelay   api.Patch[int64]  `json:"importDelay,omitempty" doc:"Seconds; null resets it to its default"`
	On            *bool             `json:"on,omitempty"`
}

// edit is e parsed against kind, the kind the client has after it.
func (e clientEdit) edit(kind clients.Kind) (downloadclients.Edit, error) {
	out := downloadclients.Edit{Name: e.Name, Username: e.Username, Folder: e.Folder, On: e.On}
	if e.NodeID != nil {
		out.Node = new(nodeOf(*e.NodeID))
	}
	if e.Kind != nil {
		out.Kind = &kind
	}
	if e.Password != "" {
		out.Password = &e.Password
	}
	if e.APIKey != "" {
		out.APIKey = &e.APIKey
	}
	if e.Daemon.Sent() {
		var d *clients.Daemon
		if v := e.Daemon.Value(); v != nil {
			d = &clients.Daemon{Host: v.Host, Port: v.Port}
		}
		out.Daemon = &d
	}
	if err := e.parsed(kind, &out); err != nil {
		return out, err
	}
	for _, f := range []struct {
		patch api.Patch[int64]
		parse func(*int64) (setting.Stored[int64], error)
		into  **setting.Stored[int64]
	}{
		{e.CheckEvery, downloadclients.CheckEvery.Parse, &out.CheckEvery},
		{e.ImportDelay, downloadclients.ImportDelay.Parse, &out.ImportDelay},
	} {
		if !f.patch.Sent() {
			continue
		}
		v, err := f.parse(f.patch.Value())
		if err != nil {
			return out, err
		}
		*f.into = &v
	}
	return out, nil
}

// parsed sets out's address, label and Client folders from e, parsed for
// kind.
func (e clientEdit) parsed(kind clients.Kind, out *downloadclients.Edit) error {
	if e.Address != nil {
		a, err := address(*e.Address, kind)
		if err != nil {
			return err
		}
		out.Address = &a
	}
	if e.Label != nil {
		l, err := label(*e.Label, kind)
		if err != nil {
			return err
		}
		out.Label = &l
	}
	if e.ClientFolders != nil {
		f, err := folders(*e.ClientFolders)
		if err != nil {
			return err
		}
		out.Folders = &f
	}
	return nil
}

type kind struct {
	Kind        string   `json:"kind" enum:"qbittorrent,transmission,deluge,rtorrent"`
	Name        string   `json:"name" doc:"The product's name, a new client's name by default"`
	Path        string   `json:"path" doc:"The path an address with none takes; empty for a kind with none"`
	Username    string   `json:"username" enum:"none,optional,required"`
	Password    string   `json:"password" enum:"none,optional,required"`
	APIKey      bool     `json:"apiKey" doc:"It may sign in with an API key in place of the username and password"`
	Daemon      bool     `json:"daemon" doc:"Its web UI connects to a daemon the Admin may choose"`
	Label       string   `json:"label" enum:"any,no_comma,lower_word" doc:"Which labels it takes: lower_word is a-z, 0-9, _ and - only"`
	Floor       string   `json:"floor" doc:"The oldest version SlipStream supports, such as 4.0"`
	Actions     []string `json:"actions" enum:"pause,resume,recheck,announce" nullable:"false"`
	DeletesData bool     `json:"deletesData" doc:"Its remove deletes data itself"`
}

var (
	needWords  = map[clients.Need]string{clients.NotTaken: "none", clients.Optional: "optional", clients.Required: "required"}
	labelWords = map[clients.LabelRule]string{clients.AnyLabel: "any", clients.NoComma: "no_comma", clients.LowerWord: "lower_word"}
)

func kindsOf() []kind {
	out := make([]kind, len(clients.Kinds))
	for i, k := range clients.Kinds {
		f := k.Facts()
		out[i] = kind{
			Kind: string(k), Name: f.Name, Path: f.Path, Username: needWords[f.SignIn.Username], Password: needWords[f.SignIn.Password],
			APIKey: f.SignIn.APIKey, Daemon: f.Daemon, Label: labelWords[f.Label], Floor: f.Floor.String(), Actions: []string{},
			DeletesData: f.DeletesData,
		}
		for _, a := range f.Acts {
			out[i].Actions = append(out[i].Actions, actWords[a])
		}
	}
	return out
}

type pollProblem struct {
	Cause    string   `json:"cause" doc:"Why the client isn't responding, refused the sign-in or needs attention"`
	Sentence string   `json:"sentence" doc:"What happened and what to do, as a sentence"`
	Fields   []string `json:"fields" nullable:"false" doc:"The form fields an Admin changes to fix it"`
}

type externalClient struct {
	ID             string                    `json:"id" format:"uuid"`
	NodeID         string                    `json:"nodeId" format:"uuid"`
	Kind           string                    `json:"kind" enum:"qbittorrent,transmission,deluge,rtorrent"`
	Name           string                    `json:"name"`
	Address        string                    `json:"address" doc:"The whole address, with the kind's usual path filled in"`
	WebUI          *string                   `json:"webUi" doc:"The client's own web UI; null for a kind with none"`
	Username       string                    `json:"username" doc:"Empty when it signs in with none"`
	PasswordSaved  bool                      `json:"passwordSaved" doc:"A password is saved. Secrets are never returned"`
	APIKeySaved    bool                      `json:"apiKeySaved" doc:"An API key is saved. Secrets are never returned"`
	Label          string                    `json:"label"`
	Folder         string                    `json:"folder" doc:"The download folder as the client sees it; empty for the client's default"`
	ClientFolders  []clientFolder            `json:"clientFolders" nullable:"false"`
	Daemon         api.Nullable[daemon]      `json:"daemon" doc:"Deluge's daemon; null for the only one and for every other kind"`
	CheckEvery     setting.DurationField     `json:"checkEvery"`
	ImportDelay    setting.DurationField     `json:"importDelay"`
	On             bool                      `json:"on"`
	Version        int64                     `json:"settingsVersion" minimum:"1" doc:"Bumped on every edit"`
	Reach          string                    `json:"reach" enum:"responding,not_responding,sign_in_refused,needs_attention,off,node_offline,unchecked"`
	Away           bool                      `json:"away" doc:"It isn't responding, refused the sign-in or needs attention while its Node is in step, so its card is red"`
	Since          *time.Time                `json:"since" doc:"When its poll state began; null before its Node first polled it"`
	ProductVersion string                    `json:"productVersion" doc:"The product's version at the last sign-in, such as 4.1.1; empty before one"`
	Problem        api.Nullable[pollProblem] `json:"problem" doc:"Why its last poll failed; null while it responds and before its first poll"`
	Torrents       int                       `json:"torrents" minimum:"0" doc:"Its live Torrents"`
	Actions        []string                  `json:"actions" enum:"pause,resume,recheck,announce" nullable:"false" doc:"The actions its kind supports"`
}

var actWords = map[clients.Act]string{clients.Pause: "pause", clients.Resume: "resume", clients.Recheck: "recheck", clients.Announce: "announce"}

func clientOf(e downloadclients.ExternalCard, node string) externalClient {
	out := externalClient{
		ID: e.ID.String(), NodeID: e.Node.String(), Kind: string(e.Kind), Name: e.Name, Address: e.Address.String(),
		Username: e.Username, PasswordSaved: e.Saved.Password, APIKeySaved: e.Saved.APIKey, Label: e.Label.String(),
		Folder: e.Folder, ClientFolders: []clientFolder{}, CheckEvery: e.CheckEveryField(), ImportDelay: e.ImportDelayField(),
		On: e.On, Version: e.Version, Reach: string(e.Reach), Away: e.Reach.Away(), Torrents: e.Torrents, Actions: []string{},
	}
	if ui, ok := e.Address.WebUI(); ok {
		out.WebUI = &ui
	}
	for _, f := range e.Folders.Pairs() {
		out.ClientFolders = append(out.ClientFolders, clientFolder{Client: f.Client, Node: f.Node})
	}
	if d := e.Daemon; d != nil {
		out.Daemon = api.Some(daemon{Host: d.Host, Port: d.Port})
	}
	if p := e.Poll; p != nil {
		out.Since, out.ProductVersion = &p.Since, p.Version
		if pr := p.Problem; pr != nil {
			out.Problem = api.Some(pollProblem{Cause: string(pr.Cause), Sentence: downloadclients.Says(*pr, e.Subject(), node), Fields: fieldsOf(pr.Cause, e.Kind)})
		}
	}
	for _, a := range e.Kind.Facts().Acts {
		out.Actions = append(out.Actions, actWords[a])
	}
	return out
}

type testReport struct {
	Product  string   `json:"product" doc:"The product's name as it responded, such as Transmission"`
	Version  string   `json:"version" doc:"Such as 4.1.1"`
	Folder   string   `json:"nodeFolder" doc:"The download folder as the Node sees it"`
	Warnings []string `json:"warnings" nullable:"false" doc:"What could go wrong with these settings, as sentences"`
	Daemons  []daemon `json:"daemons" nullable:"false" doc:"The daemons Deluge's web UI lists; empty for every other kind"`
}

func reportOf(r downloadclients.TestReport) testReport {
	out := testReport{Product: r.Product, Version: r.Version, Folder: r.NodeFolder, Warnings: []string{}}
	for _, w := range r.Warnings {
		out.Warnings = append(out.Warnings, downloadclients.Warns(w, r.Subject))
	}
	out.Daemons = daemonsOf(r.Daemons)
	return out
}

type newClientInput struct{ Body newClient }

type editClientInput struct {
	ID   string `path:"id" format:"uuid"`
	Body clientEdit
}

type testClientInput struct {
	ID   string `path:"id" format:"uuid"`
	Body *clientEdit
}

type clientOutput struct{ Body externalClient }

type testOutput struct{ Body testReport }

func clientID(raw string) downloadclients.ClientID {
	u, _ := storage.ParseUUID(raw)
	return downloadclients.ClientID{UUID: u}
}

func (h handlers) createClient(ctx context.Context, _ identity.Admin, in *newClientInput) (*clientOutput, error) {
	dc, roster := open(h.clients)
	names, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	kind := clients.Kind(in.Body.Kind)
	d, err := in.Body.draft()
	if err != nil {
		return nil, answer(err, names, kind)
	}
	e, err := dc.CreateClient(ctx, d)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	return &clientOutput{Body: clientOf(e, names[e.Node].Name)}, nil
}

func (h handlers) testDraft(ctx context.Context, _ identity.Admin, in *newClientInput) (*testOutput, error) {
	dc, roster := open(h.clients)
	names, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	kind := clients.Kind(in.Body.Kind)
	d, err := in.Body.draft()
	if err != nil {
		return nil, answer(err, names, kind)
	}
	r, err := dc.TestDraft(ctx, d)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	return &testOutput{Body: reportOf(r)}, nil
}

// edited is client id's saved kind, and body parsed against the kind the
// client has after it.
func edited(ctx context.Context, dc *downloadclients.Clients, id downloadclients.ClientID, body clientEdit) (clients.Kind, downloadclients.Edit, error) {
	was, err := dc.External(ctx, id)
	if err != nil {
		return "", downloadclients.Edit{}, err
	}
	kind := was.Kind
	if body.Kind != nil {
		kind = clients.Kind(*body.Kind)
	}
	e, err := body.edit(kind)
	return kind, e, err
}

func (h handlers) editClient(ctx context.Context, _ identity.Admin, in *editClientInput) (*clientOutput, error) {
	dc, roster := open(h.clients)
	names, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	id := clientID(in.ID)
	kind, e, err := edited(ctx, dc, id, in.Body)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	got, err := dc.EditClient(ctx, id, e)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	return &clientOutput{Body: clientOf(got, names[got.Node].Name)}, nil
}

func (h handlers) testSaved(ctx context.Context, _ identity.Admin, in *testClientInput) (*testOutput, error) {
	dc, roster := open(h.clients)
	names, err := roster.Names(ctx)
	if err != nil {
		return nil, err
	}
	var body clientEdit
	if in.Body != nil {
		body = *in.Body
	}
	id := clientID(in.ID)
	kind, e, err := edited(ctx, dc, id, body)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	r, err := dc.TestSaved(ctx, id, e)
	if err != nil {
		return nil, answer(err, names, kind)
	}
	return &testOutput{Body: reportOf(r)}, nil
}

type deleteClientInput struct {
	ID           string `path:"id" format:"uuid"`
	StopTracking bool   `query:"stopTracking" doc:"Delete a client that holds torrents SlipStream tracks: they stay in the client, and Grabs not yet imported from them fail"`
}

func (h handlers) deleteClient(ctx context.Context, _ identity.Admin, in *deleteClientInput) (*struct{}, error) {
	dc, roster := open(h.clients)
	err := dc.DeleteClient(ctx, clientID(in.ID), in.StopTracking)
	if err == nil {
		return nil, nil
	}
	names, nerr := roster.Names(ctx)
	if nerr != nil {
		return nil, nerr
	}
	return nil, answer(err, names, "")
}
