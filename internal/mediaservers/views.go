package mediaservers

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Kind lists the kinds mediaserverlink.Kinds registers, so the document's
// kind enum follows that one list.
type Kind struct{}

func (Kind) Values() []string {
	names := mediaserverlink.Kinds.Names()
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = string(n)
	}
	return out
}

// KindOf is n as the wire carries it.
func KindOf(n link.KindName) api.Enum[Kind] { return api.NewEnum[Kind](string(n)) }

type KindView struct {
	Kind    api.Enum[Kind] `json:"kind"`
	Label   string         `json:"label" doc:"The product's name, such as Jellyfin"`
	SignIn  string         `json:"signIn" enum:"plex_pin,api_key_or_admin" doc:"plex_pin signs in with a Plex account; api_key_or_admin takes an API key or an administrator's name and password"`
	Browser bool           `json:"browser" doc:"The kind takes a browser address, which Open in uses"`
	Floor   string         `json:"floor" doc:"The oldest version SlipStream supports, such as 12.2"`
}

type NodeRef struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}

type Unhealthy struct {
	Reason string    `json:"reason" enum:"not_responding,sends_failing,sign_in_refused,other_server"`
	Since  time.Time `json:"since"`
}

// State is a server's one line on the list, leading with what to do.
type State struct {
	Level string `json:"level" enum:"ok,warn,fail"`
	Words string `json:"words" doc:"The line a person reads, such as Living room Plex hasn't responded to Den since 14:02."`
}

type ServerView struct {
	ID                string          `json:"id" format:"uuid"`
	Kind              api.Enum[Kind]  `json:"kind"`
	Name              string          `json:"name"`
	Address           string          `json:"address" doc:"The address the Connect from Node uses, such as http://localhost:8096"`
	FallbackAddress   string          `json:"fallbackAddress,omitempty" doc:"The address the Controller uses while the Connect from Node is offline; absent when it uses address"`
	BrowserAddress    string          `json:"browserAddress,omitempty" doc:"The address the Open in button opens, for Jellyfin and Emby; absent when there is no Open in button"`
	Version           string          `json:"version" doc:"As the server last reported it"`
	Identity          string          `json:"identity" doc:"Plex's machineIdentifier, or the Jellyfin or Emby server Id"`
	ReachingNode      NodeRef         `json:"reachingNode" doc:"The Node that sends changes and watches sessions"`
	RunsOnNode        *NodeRef        `json:"runsOnNode,omitempty" doc:"The Node the server runs on; absent when it runs on no Node"`
	PausesConversions bool            `json:"pausesConversions" doc:"Whether Conversions on the Node it runs on pause while it transcodes video"`
	Fallback          string          `json:"fallback" enum:"available,none,untested" doc:"Whether the Controller can connect to the server itself, which it does while the Connect from Node is offline"`
	CheckedAt         *time.Time      `json:"checkedAt,omitempty" doc:"When both routes were last tested"`
	Contact           string          `json:"contact" enum:"unknown,responding,not_responding,sign_in_refused,other_server" doc:"As the Connect from Node last reported it; unknown until it reports and while it is offline"`
	ContactSince      *time.Time      `json:"contactSince,omitempty"`
	Sends             string          `json:"sends" enum:"ok,failing"`
	LastContactAt     *time.Time      `json:"lastContactAt,omitempty"`
	LastSendAt        *time.Time      `json:"lastSendAt,omitempty"`
	Unhealthy         *Unhealthy      `json:"unhealthy,omitempty" doc:"Set while the server is unhealthy"`
	State             State           `json:"state"`
	MappingsNeeded    int             `json:"mappingsNeeded" minimum:"0" doc:"Roots with shown true and no confirmed Path mapping"`
	PlexPass          *bool           `json:"plexPass,omitempty" doc:"Plex only: whether the signed-in account has Plex Pass"`
	MembersSee        bool            `json:"membersSee" doc:"Show to Members: whether Members may be offered access to the server"`
	MemberLibraries   []LibraryChoice `json:"memberLibraries" nullable:"false" doc:"The Libraries for Members: what a Member's access covers"`
	MemberAddress     string          `json:"memberAddress,omitempty" doc:"Jellyfin and Emby: the address How to watch gives Members; absent when the Admin set none"`
	MemberCount       int             `json:"memberCount" minimum:"0" doc:"How many Members can watch on the server"`
}

// Draft is a server to add, parsed from the request.
type Draft struct {
	Kind            link.KindName
	Name            string
	Address         string
	FallbackAddress string
	BrowserAddress  string
	Reaching        role.NodeID
	RunsOn          *role.NodeID
	// APIKey and DeviceID are Jellyfin's and Emby's.
	APIKey   string
	DeviceID string
	// PlexServer is the machineIdentifier the Admin picked from
	// PlexServers.
	PlexServer string
}

// Patch is a change to a saved server. A field not sent leaves the value
// alone. Null clears a nullable one.
type Patch struct {
	Name              *string
	Address           *string
	FallbackAddress   *Set[string]
	BrowserAddress    *Set[string]
	Reaching          *role.NodeID
	RunsOn            *Set[role.NodeID]
	PausesConversions *bool
	APIKey            *string
	// MembersSee on with no library is ErrNoMemberLibraries. The first
	// turn on writes the default Libraries for Members.
	MembersSee *bool
	// MemberLibraries is never empty while MembersSee is on.
	MemberLibraries *[]LibraryChoice
	// MemberAddress is refused on Plex.
	MemberAddress *Set[string]
}

// Set is a nullable field a Patch changes. A nil To clears it.
type Set[T any] struct{ To *T }

type Library struct {
	Key    string `json:"key" doc:"Plex's section key, or the Jellyfin or Emby library ItemId"`
	Name   string `json:"name"`
	Folder string `json:"folder" doc:"The library folder the mapping's folder sits inside, as the server names it"`
}

type LibraryView struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Folders []string `json:"folders" nullable:"false" doc:"Each library folder as the server names it"`
}

type MappingView struct {
	RootID       string   `json:"rootId" format:"uuid"`
	RootName     string   `json:"rootName" doc:"The root's display name, such as Movies library root on Media server"`
	RootNode     NodeRef  `json:"rootNode" doc:"The Storage Node whose disk holds the root"`
	Shown        bool     `json:"shown" doc:"false for a root that isn't in one of this server's libraries, which gets no changes"`
	ServerFolder string   `json:"serverFolder,omitempty" doc:"The path the server uses for the root's folder; absent when shown is false"`
	Library      *Library `json:"library,omitempty" doc:"The library the folder sits inside; absent when shown is false"`
	Guess        bool     `json:"guess" doc:"SlipStream pre-filled the folder from shared trailing folder names"`
	Confirmed    bool     `json:"confirmed" doc:"false while a guess or after the root was re-homed, and changes under it wait"`
	Scanned      bool     `json:"scanned" doc:"The server has content under the folder, or a full scan of its library ended"`
	Words        string   `json:"words,omitempty" doc:"What to do about the mapping, such as Living room Plex won't get changes for files in Movies (D:)."`
}

type Libraries struct {
	Libraries []LibraryView `json:"libraries" nullable:"false"`
	Mappings  []MappingView `json:"mappings" nullable:"false" doc:"Every library root, pre-filled where it has no saved mapping"`
}

type ChangeView struct {
	ID        string     `json:"id" format:"uuid"`
	Kind      string     `json:"kind" enum:"added,replaced,removed,moved,sidecar_added,sidecar_replaced,sidecar_removed,library"`
	Path      string     `json:"path" doc:"The file's path in its root; empty for a whole library"`
	FromPath  string     `json:"fromPath,omitempty" doc:"Where a moved or renamed file was"`
	State     string     `json:"state" enum:"prepared,due,sending,sent"`
	Step      string     `json:"step" enum:"none,snapshotted,scanned,source_scanned,confirmed,idle,reported,settled,old_removed,new_added,new_found,restored"`
	Result    string     `json:"result,omitempty" enum:"accepted,covered,converted" doc:"Set once sent"`
	Lost      []string   `json:"lost" nullable:"false" doc:"What a move or rename couldn't carry to the new item" enum:"watch_state,play_count,last_played,rating,collections"`
	CreatedAt time.Time  `json:"createdAt"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	Attempts  int        `json:"attempts" minimum:"0"`
	Words     string     `json:"words" doc:"The change as a person reads it, with its result or its last failure"`
}

type Changes struct {
	Changes []ChangeView `json:"changes" nullable:"false" doc:"Newest first"`
	Next    string       `json:"next,omitempty" doc:"The cursor of the next page; absent on the last"`
}

type SessionView struct {
	PlayID    string        `json:"playId" format:"uuid"`
	User      string        `json:"user"`
	Player    string        `json:"player"`
	Product   string        `json:"product"`
	Path      string        `json:"path" doc:"The file as the server names it"`
	Work      *PlayWorkView `json:"work,omitempty" doc:"The library file that plays; absent when no mapping leads to one"`
	Decision  string        `json:"decision" enum:"direct_play,direct_stream,transcode"`
	Video     string        `json:"video" enum:"direct,transcode,none"`
	Audio     string        `json:"audio" enum:"direct,transcode,none"`
	Subtitles string        `json:"subtitles" enum:"none,direct,burned"`
	Hardware  bool          `json:"hardware" doc:"The server reports hardware transcoding"`
	Reasons   []string      `json:"reasons" nullable:"false" doc:"Transcode reasons as the server gave them; Plex gives none"`
	StartedAt time.Time     `json:"startedAt"`
}

type Sessions struct {
	Reachable bool          `json:"reachable" doc:"false while the Connect from Node is offline, when sessions is empty"`
	Sessions  []SessionView `json:"sessions" nullable:"false"`
}

type HolderView struct {
	Node      NodeRef   `json:"node"`
	FetchedAt time.Time `json:"fetchedAt"`
	Current   bool      `json:"current" doc:"This Node is the server's Connect from Node now"`
}

type RouteTest struct {
	OK    bool   `json:"ok"`
	Words string `json:"words" doc:"What the test found, such as Den connected to Living room Plex."`
}

type Tested struct {
	Reaching   RouteTest `json:"reaching" doc:"From the Connect from Node at address"`
	Controller RouteTest `json:"controller" doc:"From the Controller at the fallback address"`
	Fallback   string    `json:"fallback" enum:"available,none,untested"`
}

// SignInDraft is a Jellyfin or Emby administrator's sign-in, for a server
// not yet saved.
type SignInDraft struct {
	Kind     link.KindName
	Address  string
	Node     role.NodeID
	User     string
	Password string
}

type SignedIn struct {
	APIKey   string `json:"apiKey" doc:"The key named SlipStream the sign-in created; send it as the Draft's apiKey"`
	DeviceID string `json:"deviceId" doc:"The DeviceId the sign-in used; send it as the Draft's deviceId"`
}

type PlexPin struct {
	ID        int64     `json:"pinId"`
	AuthURL   string    `json:"authUrl" format:"uri" doc:"Open it in a new tab, where the Admin signs in to Plex"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type PlexPinState struct {
	State string `json:"state" enum:"waiting,expired,signed_in,other_account" doc:"other_account is a sign-in as a Plex account other than the one SlipStream is signed in to, which replacePlexAccount confirms"`
}

type AccountView struct {
	Name       string     `json:"name" doc:"The Plex account's name"`
	PlexPass   bool       `json:"plexPass"`
	SignedInAt time.Time  `json:"signedInAt"`
	ExpiresAt  time.Time  `json:"expiresAt" doc:"When the sign-in expires unless renewed"`
	RenewedAt  *time.Time `json:"renewedAt,omitempty"`
	Refused    bool       `json:"refused" doc:"plex.tv refused the last renewal. Sign in with Plex again."`
}

type Reached struct {
	NodeID string `json:"nodeId" format:"uuid"`
	OK     bool   `json:"ok"`
}

type PlexConnection struct {
	URI         string    `json:"uri" format:"uri"`
	Local       bool      `json:"local"`
	HTTPS       bool      `json:"https"`
	ReachedFrom []Reached `json:"reachedFrom" nullable:"false" doc:"The Controller and each Node with the Media Server link Role"`
}

type PlexServerView struct {
	MachineIdentifier string           `json:"machineIdentifier"`
	Name              string           `json:"name"`
	Version           string           `json:"version"`
	Connections       []PlexConnection `json:"connections" nullable:"false" doc:"Every connection but relays"`
	SuggestedRunsOn   *NodeRef         `json:"suggestedRunsOn,omitempty" doc:"The Node whose address a local connection has"`
	SuggestedReaching NodeRef          `json:"suggestedReaching"`
}

// accessWorld is every access row with what their views read, in one
// Read.
type accessWorld struct {
	rows []db.MediaserversMemberAccess
	// servers are in the Admin's order: by name.
	servers   []db.MediaserversServer
	libraries map[link.ServerID][]LibraryChoice
	who       map[identity.AccountID]identity.Contacted
	plex      map[identity.AccountID]db.MediaserversMemberPlex
	plexPass  bool
}

func (m *MediaServers) accessWorld(ctx context.Context, r storage.Reader[storage.Library]) (accessWorld, error) {
	q := db.New(r)
	var w accessWorld
	var err error
	if w.rows, err = q.ListAccess(ctx); err != nil {
		return w, err
	}
	if w.servers, err = q.ListServers(ctx); err != nil {
		return w, err
	}
	libs, err := q.ListAllMemberLibraries(ctx)
	if err != nil {
		return w, err
	}
	w.libraries = map[link.ServerID][]LibraryChoice{}
	for _, l := range libs {
		w.libraries[l.ServerID] = append(w.libraries[l.ServerID], LibraryChoice{Key: l.LibraryKey, Name: l.Name})
	}
	plex, err := q.ListMemberPlex(ctx)
	if err != nil {
		return w, err
	}
	w.plex = map[identity.AccountID]db.MediaserversMemberPlex{}
	for _, p := range plex {
		w.plex[p.AccountID] = p
	}
	var ids []identity.AccountID
	for _, a := range w.rows {
		for _, id := range []identity.AccountID{a.AccountID, a.OfferedBy} {
			if !id.IsZero() && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	if w.who, err = identity.Contacts(ctx, r, ids); err != nil {
		return w, err
	}
	admin, _, err := m.readAccount(ctx, q)
	w.plexPass = admin.plexPass
	return w, err
}

func (w accessWorld) server(id link.ServerID) db.MediaserversServer {
	i := slices.IndexFunc(w.servers, func(s db.MediaserversServer) bool { return s.ID == id })
	return w.servers[i]
}

// plexInUse is whether one of account's Plex rows is shared or active.
func (w accessWorld) plexInUse(account identity.AccountID) bool {
	return slices.ContainsFunc(w.rows, func(a db.MediaserversMemberAccess) bool {
		return a.AccountID == account && a.Way == access.WayPlexTV && (a.State == access.Shared || a.State == access.Active)
	})
}

func (m *MediaServers) actions(w accessWorld, a db.MediaserversMemberAccess, by access.By) []access.Event {
	shares, home := m.plexWorks(w.plexPass)
	return access.Actions(access.Facts{
		State: a.State, Way: a.Way, PlexHome: a.PlexHome,
		PlexInUse:  w.plexInUse(a.AccountID),
		SharesWork: shares,
		HomeWorks:  home,
	}, by)
}

func (m *MediaServers) accessServer(srv db.MediaserversServer) AccessServer {
	return AccessServer{ID: srv.ID.String(), Name: srv.Name, Kind: KindOf(srv.Kind), Access: string(m.kind(srv.Kind).About().Access)}
}

func (m *MediaServers) accessView(w accessWorld, a db.MediaserversMemberAccess) AccessView {
	srv := w.server(a.ServerID)
	v := AccessView{
		ID: a.ID.String(), Server: m.accessServer(srv), Member: AccessMember{Name: a.MemberName},
		State: a.State, NeedsYou: a.State.NeedsYou(), Actions: m.actions(w, a, access.ByAdmin),
		Libraries: []string{}, ChangedAt: a.ChangedAt.Time(),
	}
	if !a.AccountID.IsZero() {
		c := w.who[a.AccountID]
		v.Member = AccessMember{AccountID: a.AccountID.String(), Name: c.Name, Email: c.Email, Disabled: c.Disabled}
	}
	for _, l := range w.libraries[srv.ID] {
		v.Libraries = append(v.Libraries, l.Name)
	}
	switch {
	case a.Way == access.WayPlexTV:
		v.Plex = &PlexPart{PlexHome: a.PlexHome, ByHand: a.ByHand}
		if p := w.plex[a.AccountID]; a.PlexID != nil {
			v.Plex.Username, v.Plex.Email = deref(p.Username), deref(p.Email)
		}
	case a.ServerUserID != nil:
		v.User = &UserPart{Username: deref(a.Username)}
	}
	if a.ProblemCode != nil {
		v.Problem = &AccessProblem{Code: *a.ProblemCode, Detail: deref(a.ProblemDetail)}
	}
	return v
}

// view is row id as an Admin reads it.
func (m *MediaServers) view(ctx context.Context, id AccessID) (AccessView, error) {
	var v AccessView
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		w, err := m.accessWorld(ctx, r)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(w.rows, func(a db.MediaserversMemberAccess) bool { return a.ID == id })
		if i < 0 {
			return ErrAccessNotFound
		}
		v = m.accessView(w, w.rows[i])
		return nil
	})
	return v, err
}

func (s service) Access(ctx context.Context, f AccessFilter) (AccessList, error) {
	out := AccessList{Rows: []AccessView{}}
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		w, err := s.m.accessWorld(ctx, r)
		if err != nil {
			return err
		}
		out.PlexPass = w.plexPass
		out.PlexHomeUsed = slices.ContainsFunc(w.rows, func(a db.MediaserversMemberAccess) bool {
			return a.PlexHome && (a.State == access.Shared || a.State == access.Active)
		})
		for _, needsYou := range []bool{true, false} {
			for _, a := range w.rows {
				if a.State.NeedsYou() == needsYou && (f.Account.IsZero() || a.AccountID == f.Account) {
					out.Rows = append(out.Rows, s.m.accessView(w, a))
				}
			}
		}
		if f.Account.IsZero() {
			return nil
		}
		out.Offerable, err = s.m.offerable(ctx, r, w, f.Account)
		return err
	})
	return out, err
}

// offerable is the servers shown to Members that account has no access
// to yet: none for an Admin.
func (m *MediaServers) offerable(ctx context.Context, r storage.Reader[storage.Library], w accessWorld, account identity.AccountID) ([]AccessServer, error) {
	c, err := identity.Contact(ctx, r, account)
	if errors.Is(err, storage.ErrNoRows) {
		return nil, identity.ErrNoAccount
	}
	if err != nil || c.Admin {
		return nil, err
	}
	var out []AccessServer
	for _, srv := range w.servers {
		held := slices.ContainsFunc(w.rows, func(a db.MediaserversMemberAccess) bool {
			return a.ServerID == srv.ID && a.AccountID == account
		})
		if srv.MembersSee && !held {
			out = append(out, m.accessServer(srv))
		}
	}
	return out, nil
}

func (s service) Mine(ctx context.Context, me identity.Account) (MyAccess, error) {
	account := me.Profile().ID
	out := MyAccess{Rows: []MyAccessRow{}}
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		w, err := s.m.accessWorld(ctx, r)
		if err != nil {
			return err
		}
		admins, err := identity.EnabledAdminAddresses(ctx, r)
		if err != nil {
			return err
		}
		settings, err := mail.ReadSettings(ctx, r)
		if err != nil {
			return err
		}
		out.MailSetUp = settings.Host != ""
		p := w.plex[account]
		if p.PlexID != nil {
			out.Plex = &MyPlex{Username: deref(p.Username)}
		}
		for _, srv := range w.servers {
			for _, a := range w.rows {
				if a.ServerID != srv.ID || a.AccountID != account || !a.State.MemberSees() {
					continue
				}
				out.Rows = append(out.Rows, s.m.myRow(w, a, srv, p, admins))
				out.PlexPassLine = out.PlexPassLine || (a.Way == access.WayPlexTV && !w.plexPass)
			}
		}
		return nil
	})
	return out, err
}

func (m *MediaServers) myRow(w accessWorld, a db.MediaserversMemberAccess, srv db.MediaserversServer, p db.MediaserversMemberPlex, admins []mail.Address) MyAccessRow {
	about := m.kind(srv.Kind).About()
	row := MyAccessRow{
		ID: a.ID.String(), Server: MyServer{Name: srv.Name, Label: about.Label, Access: string(about.Access)},
		AdminName: w.who[a.OfferedBy].Name, State: a.State, Pending: a.State.Pending(), Actions: m.actions(w, a, access.ByMember),
	}
	if a.OfferedBy.IsZero() && len(admins) > 0 {
		row.AdminName = admins[0].Name
	}
	if !row.Pending {
		h := ways[a.Way](m).howToWatch(a, srv, p)
		row.HowToWatch = &h
	}
	return row
}
