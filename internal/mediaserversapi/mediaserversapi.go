// Package mediaserversapi is the Media Servers' operations: the servers an
// Admin adds, their Path mappings, changes, sessions and credential
// holders, the Jellyfin and Emby administrator sign-in, and the Plex
// account. Every operation is the Admin's.
package mediaserversapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

type idInput struct {
	ID string `path:"id" format:"uuid"`
}

type draftBody struct {
	Kind            string `json:"kind" minLength:"1" doc:"One of the kinds listMediaServerKinds lists"`
	Name            string `json:"name" minLength:"1" maxLength:"100" doc:"Such as Living room Plex"`
	Address         string `json:"address,omitempty" maxLength:"2048" doc:"What the reaching Node calls, such as http://localhost:8096; absent for Plex, whose address SlipStream picks from the server's connections"`
	FallbackAddress string `json:"fallbackAddress,omitempty" maxLength:"2048" doc:"What the Controller calls; absent for address"`
	BrowserAddress  string `json:"browserAddress,omitempty" maxLength:"2048" doc:"Jellyfin and Emby: where Open in sends a browser; absent for no Open in button"`
	ReachingNodeID  string `json:"reachingNodeId" format:"uuid" doc:"The Controller, or a Node that holds Media Server link"`
	RunsOnNodeID    string `json:"runsOnNodeId,omitempty" format:"uuid" doc:"The Node the server runs on; absent when it runs on none"`
	APIKey          string `json:"apiKey,omitempty" maxLength:"1024" doc:"Jellyfin and Emby. Write-only"`
	DeviceID        string `json:"deviceId,omitempty" maxLength:"256" doc:"Jellyfin and Emby: the deviceId an administrator sign-in returned"`
	PlexServer      string `json:"plexServer,omitempty" maxLength:"256" doc:"Plex: a machineIdentifier from listPlexServers"`
}

type addInput struct{ Body draftBody }

type patchBody struct {
	Name            *string           `json:"name,omitempty" minLength:"1" maxLength:"100"`
	Address         *string           `json:"address,omitempty" minLength:"1" maxLength:"2048"`
	FallbackAddress api.Patch[string] `json:"fallbackAddress,omitempty" doc:"null for address"`
	BrowserAddress  api.Patch[string] `json:"browserAddress,omitempty" doc:"null for no Open in button"`
	ReachingNodeID  *string           `json:"reachingNodeId,omitempty" format:"uuid"`
	RunsOnNodeID    api.Patch[string] `json:"runsOnNodeId,omitempty" doc:"A Node id; null when the server runs on no Node"`
	APIKey          *string           `json:"apiKey,omitempty" minLength:"1" maxLength:"1024" doc:"Jellyfin and Emby. Write-only"`
}

type updateInput struct {
	ID   string `path:"id" format:"uuid"`
	Body patchBody
}

// MappingInput is one root's mapping to save.
type MappingInput struct {
	RootID       string `json:"rootId" format:"uuid"`
	Shown        bool   `json:"shown"`
	ServerFolder string `json:"serverFolder,omitempty" maxLength:"4096" doc:"Required when shown: the root's folder as the server sees it, such as /data/movies or D:\\Movies"`
}

type mappingsBody struct {
	Mappings []MappingInput `json:"mappings" nullable:"false" maxItems:"1000"`
}

type mappingsInput struct {
	ID   string `path:"id" format:"uuid"`
	Body mappingsBody
}

type changesInput struct {
	ID     string `path:"id" format:"uuid"`
	Cursor string `query:"cursor" maxLength:"256" doc:"The next member of the previous page; absent for the newest"`
}

type signInBody struct {
	Kind     string `json:"kind" minLength:"1"`
	Address  string `json:"address" minLength:"1" maxLength:"2048"`
	NodeID   string `json:"nodeId" format:"uuid" doc:"The reaching Node the draft names. The Controller signs in when it reaches the server, else this Node does"`
	User     string `json:"user" minLength:"1" maxLength:"256" doc:"An administrator of the server"`
	Password string `json:"password" maxLength:"1024" doc:"Write-only. Used for this one sign-in and never stored"`
}

type signInInput struct{ Body signInBody }

type pinBody struct {
	Forward string `json:"forward" format:"uri" maxLength:"2048" doc:"Where plex.tv sends the browser back to once the Admin signs in"`
}

type pinInput struct{ Body pinBody }

type pinIDInput struct {
	ID int64 `path:"id"`
}

type kindsOutput struct {
	Body struct {
		Kinds []mediaservers.KindView `json:"kinds" nullable:"false" doc:"In the order the Add sheet offers them"`
	}
}

type serversOutput struct {
	Body struct {
		Servers []mediaservers.ServerView `json:"servers" nullable:"false" doc:"By name"`
	}
}

type serverOutput struct{ Body mediaservers.ServerView }

type testedOutput struct{ Body mediaservers.Tested }

type librariesOutput struct{ Body mediaservers.Libraries }

type mappingsOutput struct {
	Body struct {
		Mappings []mediaservers.MappingView `json:"mappings" nullable:"false"`
	}
}

type sessionsOutput struct{ Body mediaservers.Sessions }

type changesOutput struct{ Body mediaservers.Changes }

type holdersOutput struct {
	Body struct {
		Holders []mediaservers.HolderView `json:"holders" nullable:"false" doc:"Every Node that fetched the server's credential, newest first. A Worker that fetched a Jellyfin or Emby key holds it until the key is replaced on the server"`
	}
}

type signedInOutput struct{ Body mediaservers.SignedIn }

type pinOutput struct{ Body mediaservers.PlexPin }

type pinStateOutput struct{ Body mediaservers.PlexPinState }

type accountOutput struct{ Body mediaservers.AccountView }

type plexServersOutput struct {
	Body struct {
		Servers []mediaservers.PlexServerView `json:"servers" nullable:"false"`
	}
}

func parseUUID(raw string, missing error) (storage.UUID, error) {
	u, err := storage.ParseUUID(raw)
	if err != nil {
		return storage.UUID{}, answer(missing)
	}
	return u, nil
}

func serverID(raw string) (link.ServerID, error) {
	u, err := parseUUID(raw, mediaservers.ErrNotFound)
	return link.ServerID{UUID: u}, err
}

func nodeID(raw string) (role.NodeID, error) {
	u, err := parseUUID(raw, mediaservers.ErrNodeNotFound)
	return role.NodeID{UUID: u}, err
}

func (b draftBody) draft() (mediaservers.Draft, error) {
	d := mediaservers.Draft{
		Kind: link.KindName(b.Kind), Name: b.Name, Address: b.Address, FallbackAddress: b.FallbackAddress,
		BrowserAddress: b.BrowserAddress, APIKey: b.APIKey, DeviceID: b.DeviceID, PlexServer: b.PlexServer,
	}
	var err error
	if d.Reaching, err = nodeID(b.ReachingNodeID); err != nil {
		return d, err
	}
	if b.RunsOnNodeID != "" {
		n, err := nodeID(b.RunsOnNodeID)
		if err != nil {
			return d, err
		}
		d.RunsOn = &n
	}
	return d, nil
}

func setOf(p api.Patch[string]) *mediaservers.Set[string] {
	if !p.Sent() {
		return nil
	}
	return &mediaservers.Set[string]{To: p.Value()}
}

func (b patchBody) patch() (mediaservers.Patch, error) {
	p := mediaservers.Patch{
		Name: b.Name, Address: b.Address, APIKey: b.APIKey,
		FallbackAddress: setOf(b.FallbackAddress), BrowserAddress: setOf(b.BrowserAddress),
	}
	if b.ReachingNodeID != nil {
		n, err := nodeID(*b.ReachingNodeID)
		if err != nil {
			return p, err
		}
		p.Reaching = &n
	}
	if b.RunsOnNodeID.Sent() {
		p.RunsOn = &mediaservers.Set[role.NodeID]{}
		if raw := b.RunsOnNodeID.Value(); raw != nil {
			n, err := nodeID(*raw)
			if err != nil {
				return p, err
			}
			p.RunsOn.To = &n
		}
	}
	return p, nil
}

func mappingsOf(in []MappingInput) ([]mediaservers.Mapping, error) {
	out := make([]mediaservers.Mapping, len(in))
	for i, m := range in {
		u, err := parseUUID(m.RootID, mediaservers.ErrRootNotFound)
		if err != nil {
			return nil, err
		}
		out[i] = mediaservers.Mapping{Root: roots.ID{UUID: u}, Shown: m.Shown, Folder: m.ServerFolder}
	}
	return out, nil
}

type handlers struct{ svc mediaservers.Service }

func (handlers) listKinds(context.Context, identity.Admin, *struct{}) (*kindsOutput, error) {
	out := &kindsOutput{}
	out.Body.Kinds = []mediaservers.KindView{}
	for _, a := range mediaserverlink.Kinds.Abouts() {
		out.Body.Kinds = append(out.Body.Kinds, mediaservers.KindView{
			Kind: mediaservers.KindOf(a.Name), Label: a.Label, SignIn: string(a.Form), Browser: a.Browser, Floor: a.Floor.String(),
		})
	}
	return out, nil
}

func (h handlers) list(ctx context.Context, _ identity.Admin, _ *struct{}) (*serversOutput, error) {
	servers, err := h.svc.List(ctx)
	if err != nil {
		return nil, answer(err)
	}
	out := &serversOutput{}
	out.Body.Servers = servers
	return out, nil
}

func (h handlers) get(ctx context.Context, _ identity.Admin, in *idInput) (*serverOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	return server(h.svc.Get(ctx, id))
}

func server(v mediaservers.ServerView, err error) (*serverOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	return &serverOutput{Body: v}, nil
}

func (h handlers) add(ctx context.Context, _ identity.Admin, in *addInput) (*serverOutput, error) {
	d, err := in.Body.draft()
	if err != nil {
		return nil, err
	}
	return server(h.svc.Add(ctx, d))
}

func (h handlers) update(ctx context.Context, _ identity.Admin, in *updateInput) (*serverOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	p, err := in.Body.patch()
	if err != nil {
		return nil, err
	}
	return server(h.svc.Update(ctx, id, p))
}

func (h handlers) remove(ctx context.Context, _ identity.Admin, in *idInput) (*struct{}, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Remove(ctx, id); err != nil {
		return nil, answer(err)
	}
	return &struct{}{}, nil
}

func (h handlers) test(ctx context.Context, _ identity.Admin, in *idInput) (*testedOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	t, err := h.svc.Test(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &testedOutput{Body: t}, nil
}

func (h handlers) confirmIdentity(ctx context.Context, _ identity.Admin, in *idInput) (*serverOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	return server(h.svc.ConfirmIdentity(ctx, id))
}

func (h handlers) libraries(ctx context.Context, _ identity.Admin, in *idInput) (*librariesOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	l, err := h.svc.Libraries(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &librariesOutput{Body: l}, nil
}

func (h handlers) saveMappings(ctx context.Context, _ identity.Admin, in *mappingsInput) (*mappingsOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	ms, err := mappingsOf(in.Body.Mappings)
	if err != nil {
		return nil, err
	}
	saved, err := h.svc.SaveMappings(ctx, id, ms)
	if err != nil {
		return nil, answer(err)
	}
	out := &mappingsOutput{}
	out.Body.Mappings = saved
	return out, nil
}

func (h handlers) refresh(ctx context.Context, _ identity.Admin, in *idInput) (*struct{}, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Refresh(ctx, id); err != nil {
		return nil, answer(err)
	}
	return &struct{}{}, nil
}

func (h handlers) sessions(ctx context.Context, _ identity.Admin, in *idInput) (*sessionsOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	s, err := h.svc.Sessions(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	return &sessionsOutput{Body: s}, nil
}

func (h handlers) changes(ctx context.Context, _ identity.Admin, in *changesInput) (*changesOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	c, err := h.svc.Changes(ctx, id, in.Cursor)
	if err != nil {
		return nil, answer(err)
	}
	return &changesOutput{Body: c}, nil
}

func (h handlers) holders(ctx context.Context, _ identity.Admin, in *idInput) (*holdersOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	hs, err := h.svc.CredentialHolders(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	out := &holdersOutput{}
	out.Body.Holders = hs
	return out, nil
}

func (h handlers) signIn(ctx context.Context, _ identity.Admin, in *signInInput) (*signedInOutput, error) {
	n, err := nodeID(in.Body.NodeID)
	if err != nil {
		return nil, err
	}
	s, err := h.svc.SignIn(ctx, mediaservers.SignInDraft{
		Kind: link.KindName(in.Body.Kind), Address: in.Body.Address, Node: n, User: in.Body.User, Password: in.Body.Password,
	})
	if err != nil {
		return nil, answer(err)
	}
	return &signedInOutput{Body: s}, nil
}

func (h handlers) startPin(ctx context.Context, _ identity.Admin, in *pinInput) (*pinOutput, error) {
	p, err := h.svc.StartPlexPin(ctx, in.Body.Forward)
	if err != nil {
		return nil, answer(err)
	}
	return &pinOutput{Body: p}, nil
}

func (h handlers) pin(ctx context.Context, _ identity.Admin, in *pinIDInput) (*pinStateOutput, error) {
	s, err := h.svc.PlexPin(ctx, in.ID)
	if err != nil {
		return nil, answer(err)
	}
	return &pinStateOutput{Body: s}, nil
}

func (h handlers) account(ctx context.Context, _ identity.Admin, _ *struct{}) (*accountOutput, error) {
	a, err := h.svc.PlexAccount(ctx)
	if err != nil {
		return nil, answer(err)
	}
	return &accountOutput{Body: a}, nil
}

func (h handlers) signOut(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
	if err := h.svc.SignOutOfPlex(ctx); err != nil {
		return nil, answer(err)
	}
	return &struct{}{}, nil
}

func (h handlers) plexServers(ctx context.Context, _ identity.Admin, _ *struct{}) (*plexServersOutput, error) {
	ss, err := h.svc.PlexServers(ctx)
	if err != nil {
		return nil, answer(err)
	}
	out := &plexServersOutput{}
	out.Body.Servers = ss
	return out, nil
}

// Routes registers the Media Servers' operations against svc.
func Routes(r *api.Router, svc mediaservers.Service) {
	o := r.Owner("mediaservers", api.Admin)
	h := handlers{svc}
	const base = "/media-servers"
	saved := []*api.Code{tooOld, otherServer, unreached, signInRefused, localhostOnWorker, nameTaken, nodeNotFound, nodeNotLinked}
	add := append([]*api.Code{kindUnknown, plexNotSignedIn, plexServerNotFound, plexTVUnreached, notBuilt}, saved...)
	update := append([]*api.Code{serverNotFound, notBuilt}, saved...)
	plex := []*api.Code{plexNotSignedIn, plexTVUnreached, notBuilt}

	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerKinds", Method: http.MethodGet, Path: base + "/kinds", Status: http.StatusOK,
		Summary: "List the kinds of Media Server SlipStream supports, with the sign-in each takes and its oldest supported version",
	}, h.listKinds)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServers", Method: http.MethodGet, Path: base, Status: http.StatusOK,
		Summary: "List the Media Servers SlipStream keeps up to date, each with its state",
	}, h.list)
	api.Register(o, api.Admin, api.Op{
		ID: "addMediaServer", Method: http.MethodPost, Path: base, Status: http.StatusCreated,
		Summary: "Add a Media Server once it responds from its reaching Node and accepts SlipStream's sign-in",
		Errors:  add,
	}, h.add)
	api.Register(o, api.Admin, api.Op{
		ID: "getMediaServer", Method: http.MethodGet, Path: base + "/{id}", Status: http.StatusOK,
		Summary: "Get a Media Server with its state",
		Errors:  []*api.Code{serverNotFound},
	}, h.get)
	api.Register(o, api.Admin, api.Op{
		ID: "updateMediaServer", Method: http.MethodPatch, Path: base + "/{id}", Status: http.StatusOK,
		Summary: "Change a Media Server, testing it again when the change is to its address, reaching Node or API key",
		Errors:  update,
	}, h.update)
	api.Register(o, api.Admin, api.Op{
		ID: "removeMediaServer", Method: http.MethodDelete, Path: base + "/{id}", Status: http.StatusNoContent,
		Summary: "Remove a Media Server, which also deletes its queued changes, its path index and its play history",
		Errors:  []*api.Code{serverNotFound, notBuilt},
	}, h.remove)
	api.Register(o, api.Admin, api.Op{
		ID: "testMediaServer", Method: http.MethodPost, Path: base + "/{id}/test", Status: http.StatusOK,
		Summary: "Test a Media Server from its reaching Node at its address and from the Controller at its fallback address",
		Errors:  []*api.Code{serverNotFound, unreached, notBuilt},
	}, h.test)
	api.Register(o, api.Admin, api.Op{
		ID: "confirmMediaServerIdentity", Method: http.MethodPost, Path: base + "/{id}/confirm-identity", Status: http.StatusOK,
		Summary: "Accept the server that responds at a Media Server's address now as that Media Server",
		Errors:  []*api.Code{serverNotFound, unreached, tooOld, signInRefused, notBuilt},
	}, h.confirmIdentity)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerLibraries", Method: http.MethodGet, Path: base + "/{id}/libraries", Status: http.StatusOK,
		Summary: "List a Media Server's library folders and every library root's Path mapping, pre-filled where none is saved",
		Errors:  []*api.Code{serverNotFound, unreached, signInRefused, notBuilt},
	}, h.libraries)
	api.Register(o, api.Admin, api.Op{
		ID: "saveMediaServerMappings", Method: http.MethodPut, Path: base + "/{id}/mappings", Status: http.StatusOK,
		Summary: "Save a Media Server's Path mappings once each shown folder sits inside one of its library folders",
		Errors:  []*api.Code{serverNotFound, rootNotFound, outsideLibraries, unreached, signInRefused, notBuilt},
	}, h.saveMappings)
	api.Register(o, api.Admin, api.Op{
		ID: "refreshMediaServer", Method: http.MethodPost, Path: base + "/{id}/refresh", Status: http.StatusAccepted,
		Summary: "Queue a whole library refresh of every library a Media Server's confirmed Path mappings lead into",
		Errors:  []*api.Code{serverNotFound, notBuilt},
	}, h.refresh)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerSessions", Method: http.MethodGet, Path: base + "/{id}/sessions", Status: http.StatusOK,
		Summary: "List what plays on a Media Server now, as its reaching Node last saw it",
		Errors:  []*api.Code{serverNotFound, notBuilt},
	}, h.sessions)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerChanges", Method: http.MethodGet, Path: base + "/{id}/changes", Status: http.StatusOK,
		Summary: "List a Media Server's recent changes, newest first, with their results and what each move couldn't carry",
		Errors:  []*api.Code{serverNotFound, notBuilt},
	}, h.changes)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerCredentialHolders", Method: http.MethodGet, Path: base + "/{id}/credential-holders", Status: http.StatusOK,
		Summary: "List every Node that fetched a Media Server's credential",
		Errors:  []*api.Code{serverNotFound, notBuilt},
	}, h.holders)
	api.Register(o, api.Admin, api.Op{
		ID: "signInToMediaServer", Method: http.MethodPost, Path: base + "/sign-in", Status: http.StatusOK,
		Summary: "Sign in to a Jellyfin or Emby server as an administrator once, creating an API key named SlipStream for a draft server",
		Errors:  []*api.Code{kindUnknown, signInRefused, unreached, tooOld, nodeNotFound, nodeNotLinked, notBuilt},
	}, h.signIn)
	api.Register(o, api.Admin, api.Op{
		ID: "startPlexSignIn", Method: http.MethodPost, Path: base + "/plex/pin", Status: http.StatusCreated,
		Summary: "Start a Plex sign-in and get the plex.tv page the Admin signs in on",
		Errors:  []*api.Code{plexTVUnreached, notBuilt},
	}, h.startPin)
	api.Register(o, api.Admin, api.Op{
		ID: "getPlexSignIn", Method: http.MethodGet, Path: base + "/plex/pin/{id}", Status: http.StatusOK,
		Summary: "Get whether a Plex sign-in is still waiting, has expired or signed in",
		Errors:  []*api.Code{plexPinNotFound, notBuilt},
	}, h.pin)
	api.Register(o, api.Admin, api.Op{
		ID: "getPlexAccount", Method: http.MethodGet, Path: base + "/plex/account", Status: http.StatusOK,
		Summary: "Get the Plex account SlipStream is signed in to and when its sign-in expires",
		Errors:  []*api.Code{plexNotSignedIn, notBuilt},
	}, h.account)
	api.Register(o, api.Admin, api.Op{
		ID: "signOutOfPlex", Method: http.MethodDelete, Path: base + "/plex/account", Status: http.StatusNoContent,
		Summary: "Sign out of Plex, which also removes every Plex server",
		Errors:  []*api.Code{notBuilt},
	}, h.signOut)
	api.Register(o, api.Admin, api.Op{
		ID: "listPlexServers", Method: http.MethodGet, Path: base + "/plex/servers", Status: http.StatusOK,
		Summary: "List the Plex account's servers and which Nodes reach each of their connections",
		Errors:  append([]*api.Code{signInRefused}, plex...),
	}, h.plexServers)
}
