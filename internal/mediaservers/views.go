package mediaservers

import (
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/role"
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
	ID              string         `json:"id" format:"uuid"`
	Kind            api.Enum[Kind] `json:"kind"`
	Name            string         `json:"name"`
	Address         string         `json:"address" doc:"What the reaching Node calls, such as http://localhost:8096"`
	FallbackAddress string         `json:"fallbackAddress,omitempty" doc:"What the Controller calls; absent when it calls address"`
	BrowserAddress  string         `json:"browserAddress,omitempty" doc:"Where Open in sends a browser, for Jellyfin and Emby; absent for no Open in button"`
	Version         string         `json:"version" doc:"As the server last reported it"`
	Identity        string         `json:"identity" doc:"Plex's machineIdentifier, or the Jellyfin or Emby server Id"`
	ReachingNode    NodeRef        `json:"reachingNode" doc:"The Node that sends changes and watches sessions"`
	RunsOnNode      *NodeRef       `json:"runsOnNode,omitempty" doc:"The Node the server runs on; absent when it runs on no Node"`
	Fallback        string         `json:"fallback" enum:"available,none,untested" doc:"Whether the Controller reaches the server itself, which it does while the reaching Node is offline"`
	CheckedAt       *time.Time     `json:"checkedAt,omitempty" doc:"When both routes were last tested"`
	Contact         string         `json:"contact" enum:"unknown,responding,not_responding,sign_in_refused,other_server" doc:"As the reaching Node last reported it; unknown until it reports and while it is offline"`
	ContactSince    *time.Time     `json:"contactSince,omitempty"`
	Sends           string         `json:"sends" enum:"ok,failing"`
	LastContactAt   *time.Time     `json:"lastContactAt,omitempty"`
	LastSendAt      *time.Time     `json:"lastSendAt,omitempty"`
	Unhealthy       *Unhealthy     `json:"unhealthy,omitempty" doc:"Set while the server is unhealthy"`
	State           State          `json:"state"`
	MappingsNeeded  int            `json:"mappingsNeeded" minimum:"0" doc:"Shown roots without a confirmed Path mapping"`
	PlexPass        *bool          `json:"plexPass,omitempty" doc:"Plex only: whether the signed-in account has Plex Pass"`
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
	Name            *string
	Address         *string
	FallbackAddress *Set[string]
	BrowserAddress  *Set[string]
	Reaching        *role.NodeID
	RunsOn          *Set[role.NodeID]
	APIKey          *string
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
	RootName     string   `json:"rootName" doc:"Such as Movies (D:)"`
	RootNode     NodeRef  `json:"rootNode" doc:"The Storage Node whose disk holds the root"`
	Shown        bool     `json:"shown" doc:"false for a root this server doesn't show, which gets no changes"`
	ServerFolder string   `json:"serverFolder,omitempty" doc:"The root's folder as the server sees it; absent when not shown"`
	Library      *Library `json:"library,omitempty" doc:"The library the folder sits inside; absent when not shown"`
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
	PlayID    string    `json:"playId" format:"uuid"`
	User      string    `json:"user"`
	Player    string    `json:"player"`
	Product   string    `json:"product"`
	Path      string    `json:"path" doc:"The file as the server names it"`
	WorkID    string    `json:"workId,omitempty" format:"uuid" doc:"The Work whose file plays; absent when no mapping leads to a library file"`
	Decision  string    `json:"decision" enum:"direct_play,copy,transcode"`
	Hardware  bool      `json:"hardware" doc:"The server reports hardware transcoding"`
	Reasons   []string  `json:"reasons" nullable:"false" doc:"Transcode reasons as the server gave them; Plex gives none"`
	StartedAt time.Time `json:"startedAt"`
}

type Sessions struct {
	Reachable bool          `json:"reachable" doc:"false while the reaching Node is offline, when sessions is empty"`
	Sessions  []SessionView `json:"sessions" nullable:"false"`
}

type HolderView struct {
	Node      NodeRef   `json:"node"`
	FetchedAt time.Time `json:"fetchedAt"`
	Current   bool      `json:"current" doc:"The server's reaching Node now"`
}

type RouteTest struct {
	OK    bool   `json:"ok"`
	Words string `json:"words" doc:"What the test found, such as Den reached Living room Plex."`
}

type Tested struct {
	Reaching   RouteTest `json:"reaching" doc:"From the reaching Node at address"`
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
	State string `json:"state" enum:"waiting,expired,signed_in"`
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
	ReachedFrom []Reached `json:"reachedFrom" nullable:"false" doc:"The Controller and each Node that holds Media Server link"`
}

type PlexServerView struct {
	MachineIdentifier string           `json:"machineIdentifier"`
	Name              string           `json:"name"`
	Version           string           `json:"version"`
	Connections       []PlexConnection `json:"connections" nullable:"false" doc:"Every connection but relays"`
	SuggestedRunsOn   *NodeRef         `json:"suggestedRunsOn,omitempty" doc:"The Node whose address a local connection has"`
	SuggestedReaching NodeRef          `json:"suggestedReaching"`
}
