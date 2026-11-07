package external

import (
	"errors"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

var (
	runOnWire       = map[clients.Run]dlv1.Wanted{clients.Started: dlv1.Wanted_WANTED_STARTED, clients.Paused: dlv1.Wanted_WANTED_PAUSED}
	pollStateOnWire = map[clients.PollState]dlv1.PollState{
		clients.Responding:     dlv1.PollState_POLL_STATE_RESPONDING,
		clients.NotResponding:  dlv1.PollState_POLL_STATE_NOT_RESPONDING,
		clients.SignInRefused:  dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED,
		clients.NeedsAttention: dlv1.PollState_POLL_STATE_NEEDS_ATTENTION,
	}
	causeOnWire = map[clients.Cause]dlv1.Cause{
		clients.NoResponse:       dlv1.Cause_CAUSE_NO_RESPONSE,
		clients.TimedOut:         dlv1.Cause_CAUSE_TIMED_OUT,
		clients.TLSFailed:        dlv1.Cause_CAUSE_TLS_FAILED,
		clients.ServerError:      dlv1.Cause_CAUSE_SERVER_ERROR,
		clients.WrongCredentials: dlv1.Cause_CAUSE_WRONG_CREDENTIALS,
		clients.WrongAPIKey:      dlv1.Cause_CAUSE_WRONG_API_KEY,
		clients.Banned:           dlv1.Cause_CAUSE_BANNED,
		clients.LockedOut:        dlv1.Cause_CAUSE_LOCKED_OUT,
		clients.VersionTooOld:    dlv1.Cause_CAUSE_VERSION_TOO_OLD,
		clients.LabelPluginOff:   dlv1.Cause_CAUSE_LABEL_PLUGIN_OFF,
		clients.FolderUnreadable: dlv1.Cause_CAUSE_FOLDER_UNREADABLE,
		clients.FolderInRoot:     dlv1.Cause_CAUSE_FOLDER_IN_ROOT,
		clients.AddressRefused:   dlv1.Cause_CAUSE_ADDRESS_REFUSED,
		clients.HostRefused:      dlv1.Cause_CAUSE_HOST_REFUSED,
		clients.DaemonNeeded:     dlv1.Cause_CAUSE_DAEMON_NEEDED,
		clients.WrongDaemon:      dlv1.Cause_CAUSE_WRONG_DAEMON,
		clients.NotKept:          dlv1.Cause_CAUSE_NOT_KEPT,
		clients.MostMissing:      dlv1.Cause_CAUSE_MOST_MISSING,
	}
)

func runOf(w dlv1.Wanted) clients.Run {
	if w == dlv1.Wanted_WANTED_PAUSED {
		return clients.Paused
	}
	return clients.Started
}

// ProblemOnWire is p as the Controller reads it, nil for none.
func ProblemOnWire(p *clients.Problem) *dlv1.Problem {
	if p == nil {
		return nil
	}
	out := &dlv1.Problem{
		Cause: causeOnWire[p.Cause], Status: int32(p.Status), Folder: p.Folder, Paired: p.Paired, //nolint:gosec // an HTTP status fits
		Host: p.Host, Missing: int32(p.Missing), //nolint:gosec // a count of torrents fits
	}
	if p.Cause == clients.VersionTooOld {
		out.Version = p.Version.String()
	}
	return out
}

// problemOf is a failed call's error as a Problem: a response the kind
// can't read is a client that isn't responding as SlipStream speaks to it.
func problemOf(err error) *clients.Problem {
	if p, ok := errors.AsType[*clients.Problem](err); ok {
		return p
	}
	return &clients.Problem{Cause: clients.NoResponse, Err: err}
}
