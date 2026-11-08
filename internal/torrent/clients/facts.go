package clients

// Kind is one product SlipStream speaks to, by its stored and wire name.
type Kind string

const (
	QBittorrent  Kind = "qbittorrent"
	Transmission Kind = "transmission"
	Deluge       Kind = "deluge"
	RTorrent     Kind = "rtorrent"
)

// Kinds is every Kind, in the order the form offers them. The schema
// CHECK, the proto enum and the Node's openers each match it.
var Kinds = []Kind{QBittorrent, Transmission, Deluge, RTorrent}

// Facts is what SlipStream knows of a kind without reaching it.
type Facts struct {
	// Name is the product's name, the default client name and the subject
	// of its sentences: "Transmission".
	Name string
	// Path is what ParseAddress fills in for an address with no path, ""
	// for a kind that has none.
	Path string
	// WebUI is the path of the client's own web UI on its address's host,
	// "" for none.
	WebUI  string
	SignIn SignIn
	// Daemon is a kind whose web UI connects to a daemon the Admin picks.
	Daemon bool
	Label  LabelRule
	// Floor is the oldest version SlipStream supports, the lowest a fake
	// plays.
	Floor Version
	Acts  []Act
	// DeletesData is false for a kind whose remove never deletes data, so
	// the Node deletes the torrent's files itself.
	DeletesData bool
	// Lockout is a kind that locks the Node out after failed sign-ins, so
	// a refused sign-in stops polling.
	Lockout bool
	// NoMovingState is a kind that can move finished files without
	// reporting it, so the Test warns while the Import delay is 0.
	NoMovingState bool
}

// SignIn is which sign-in fields a kind takes.
type SignIn struct {
	Username, Password Need
	// APIKey is a kind that may sign in with an API key in place of the
	// username and password.
	APIKey bool
}

type Need uint8

const (
	NotTaken Need = iota
	Optional
	Required
)

var allActs = []Act{Pause, Resume, Recheck, Announce}

var facts = map[Kind]Facts{
	QBittorrent: {
		Name:        "qBittorrent",
		WebUI:       "/",
		SignIn:      SignIn{Username: Optional, Password: Optional, APIKey: true},
		Label:       AnyLabel,
		Floor:       Version{4, 6, 0},
		Acts:        allActs,
		DeletesData: true,
		Lockout:     true,
	},
	Transmission: {
		Name:        "Transmission",
		Path:        "/transmission/rpc",
		WebUI:       "/transmission/web/",
		SignIn:      SignIn{Username: Optional, Password: Optional},
		Label:       NoComma,
		Floor:       Version{4, 0, 0},
		Acts:        allActs,
		DeletesData: true,
		Lockout:     true,
	},
	Deluge: {
		Name:        "Deluge",
		Path:        "/json",
		WebUI:       "/",
		SignIn:      SignIn{Password: Required},
		Daemon:      true,
		Label:       LowerWord,
		Floor:       Version{2, 1, 0},
		Acts:        allActs,
		DeletesData: true,
	},
	RTorrent: {
		Name:          "rTorrent",
		Path:          "/RPC2",
		SignIn:        SignIn{Username: Optional, Password: Optional},
		Label:         AnyLabel,
		Floor:         Version{0, 9, 8},
		Acts:          allActs,
		NoMovingState: true,
	},
}

// Facts is k's facts. Every Kind in Kinds has them.
func (k Kind) Facts() Facts { return facts[k] }
