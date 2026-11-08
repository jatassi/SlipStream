package clients

import "fmt"

// PollState is how a client's polls stand, as its Node reports it. Off
// and Node offline are read from the record and the link, never reported.
type PollState string

const (
	Responding     PollState = "responding"
	NotResponding  PollState = "not_responding"
	SignInRefused  PollState = "sign_in_refused"
	NeedsAttention PollState = "needs_attention"
)

// Cause is why a client isn't responding, refused the sign-in or needs
// attention.
type Cause string

const (
	NoResponse  Cause = "no_response"
	TimedOut    Cause = "timed_out"
	TLSFailed   Cause = "tls_failed"
	ServerError Cause = "server_error"

	WrongCredentials Cause = "wrong_credentials" //nolint:gosec // a cause's name, not a credential
	WrongAPIKey      Cause = "wrong_api_key"
	// Banned is qBittorrent's ban after failed sign-ins.
	Banned Cause = "banned"
	// LockedOut is Transmission's lockout after failed sign-ins.
	LockedOut Cause = "locked_out"

	VersionTooOld    Cause = "version_too_old"
	LabelPluginOff   Cause = "label_plugin_off"
	FolderUnreadable Cause = "folder_unreadable"
	// FolderInRoot is a download folder inside a library root, found by a
	// Test.
	FolderInRoot Cause = "folder_in_root"
	// AddressRefused is Transmission's rpc-whitelist.
	AddressRefused Cause = "address_refused"
	// HostRefused is Transmission's 421 or qBittorrent's host check.
	HostRefused Cause = "host_refused"
	// DaemonNeeded is a Test of a Deluge whose web UI lists more than one
	// daemon and none chosen.
	DaemonNeeded Cause = "daemon_needed"
	WrongDaemon  Cause = "wrong_daemon"
	NotKept      Cause = "not_kept"
	MostMissing  Cause = "most_missing"
)

// State is the poll state a cause puts its client in.
func (c Cause) State() PollState {
	switch c {
	case NoResponse, TimedOut, TLSFailed, ServerError:
		return NotResponding
	case WrongCredentials, WrongAPIKey, Banned, LockedOut:
		return SignInRefused
	case VersionTooOld, LabelPluginOff, FolderUnreadable, FolderInRoot, AddressRefused, HostRefused,
		DaemonNeeded, WrongDaemon, NotKept, MostMissing:
		return NeedsAttention
	}
	panic("clients: no poll state for cause " + string(c))
}

// Polls is whether a client in this cause still answers polls, so its
// torrents' data is current.
func (c Cause) Polls() bool { return c == NotKept || c == MostMissing }

// Field is a form field a cause names, for the web to mark.
type Field string

const (
	FieldAddress  Field = "address"
	FieldUsername Field = "username"
	FieldPassword Field = "password"
	FieldAPIKey   Field = "apiKey"
	FieldLabel    Field = "label"
	FieldFolder   Field = "folder"
	FieldFolders  Field = "clientFolders"
	FieldDaemon   Field = "daemon"
)

// Fields are the form fields an Admin changes to fix c.
func (c Cause) Fields() []Field {
	switch c {
	case NoResponse, TimedOut, TLSFailed, HostRefused, MostMissing:
		return []Field{FieldAddress}
	case WrongCredentials:
		return []Field{FieldUsername, FieldPassword}
	case WrongAPIKey:
		return []Field{FieldAPIKey}
	case LabelPluginOff:
		return []Field{FieldLabel}
	case FolderUnreadable:
		return []Field{FieldFolder, FieldFolders}
	case FolderInRoot:
		return []Field{FieldFolder}
	case DaemonNeeded, WrongDaemon:
		return []Field{FieldDaemon}
	case ServerError, Banned, LockedOut, VersionTooOld, AddressRefused, NotKept:
		return nil
	}
	panic("clients: no fields for cause " + string(c))
}

// Problem is what went wrong with a client, and the facts its sentence
// names. A kind returns it as an error.
type Problem struct {
	Cause Cause
	// Status is the HTTP status, for ServerError.
	Status int
	// Version is the version reached, for VersionTooOld.
	Version Version
	// Folder is the folder tried as the Node sees it, and Paired the
	// client folder it is paired with, "" for none.
	Folder, Paired string
	// Host is the host name refused, for HostRefused.
	Host string
	// Missing is how many torrents the client no longer lists, for
	// MostMissing.
	Missing int
	// Err is the transport error behind a not responding cause, for logs.
	Err error
}

func (p *Problem) Error() string {
	msg := p.what()
	if p.Err != nil {
		return msg + ": " + p.Err.Error()
	}
	return msg
}

func (p *Problem) Unwrap() error { return p.Err }

func (p *Problem) what() string {
	switch p.Cause {
	case NoResponse:
		return "the client didn't respond"
	case TimedOut:
		return "the client didn't respond in time"
	case TLSFailed:
		return "the TLS connection to the client failed"
	case ServerError:
		return fmt.Sprintf("the client responded with status %d", p.Status)
	case WrongCredentials:
		return "the client rejected the username or password"
	case WrongAPIKey:
		return "the client rejected the API key"
	case Banned:
		return "the client banned the Node's address after failed sign-ins"
	case LockedOut:
		return "the client locked out the Node after failed sign-ins"
	case VersionTooOld:
		return fmt.Sprintf("the client's version %s is older than SlipStream supports", p.Version)
	case LabelPluginOff:
		return "the client's Label plugin is off"
	case FolderUnreadable:
		return "the Node can't open " + p.Folder
	case FolderInRoot:
		return fmt.Sprintf("the download folder %s is inside a library root", p.Folder)
	case AddressRefused:
		return "the client refused the Node's address"
	case HostRefused:
		return "the client refused the host name " + p.Host
	case DaemonNeeded:
		return "the web UI lists more than one daemon and none is chosen"
	case WrongDaemon:
		return "the web UI is connected to another daemon"
	case NotKept:
		return "the client didn't keep a torrent SlipStream sent it"
	case MostMissing:
		return fmt.Sprintf("the client no longer lists %d of SlipStream's torrents", p.Missing)
	}
	panic("clients: no words for cause " + string(p.Cause))
}
