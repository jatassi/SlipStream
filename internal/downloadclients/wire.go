package downloadclients

import (
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

var (
	wantedFromWire = map[dlv1.Wanted]Wanted{dlv1.Wanted_WANTED_STARTED: Started, dlv1.Wanted_WANTED_PAUSED: Paused}
	kindOnWire     = map[clients.Kind]dlv1.Kind{
		clients.QBittorrent:  dlv1.Kind_KIND_QBITTORRENT,
		clients.Transmission: dlv1.Kind_KIND_TRANSMISSION,
		clients.Deluge:       dlv1.Kind_KIND_DELUGE,
		clients.RTorrent:     dlv1.Kind_KIND_RTORRENT,
	}
	pollStateFromWire = map[dlv1.PollState]clients.PollState{
		dlv1.PollState_POLL_STATE_RESPONDING:      clients.Responding,
		dlv1.PollState_POLL_STATE_NOT_RESPONDING:  clients.NotResponding,
		dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED: clients.SignInRefused,
		dlv1.PollState_POLL_STATE_NEEDS_ATTENTION: clients.NeedsAttention,
	}
	causeFromWire = map[dlv1.Cause]clients.Cause{
		dlv1.Cause_CAUSE_NO_RESPONSE:       clients.NoResponse,
		dlv1.Cause_CAUSE_TIMED_OUT:         clients.TimedOut,
		dlv1.Cause_CAUSE_TLS_FAILED:        clients.TLSFailed,
		dlv1.Cause_CAUSE_SERVER_ERROR:      clients.ServerError,
		dlv1.Cause_CAUSE_WRONG_CREDENTIALS: clients.WrongCredentials,
		dlv1.Cause_CAUSE_WRONG_API_KEY:     clients.WrongAPIKey,
		dlv1.Cause_CAUSE_BANNED:            clients.Banned,
		dlv1.Cause_CAUSE_LOCKED_OUT:        clients.LockedOut,
		dlv1.Cause_CAUSE_VERSION_TOO_OLD:   clients.VersionTooOld,
		dlv1.Cause_CAUSE_LABEL_PLUGIN_OFF:  clients.LabelPluginOff,
		dlv1.Cause_CAUSE_FOLDER_UNREADABLE: clients.FolderUnreadable,
		dlv1.Cause_CAUSE_FOLDER_IN_ROOT:    clients.FolderInRoot,
		dlv1.Cause_CAUSE_ADDRESS_REFUSED:   clients.AddressRefused,
		dlv1.Cause_CAUSE_HOST_REFUSED:      clients.HostRefused,
		dlv1.Cause_CAUSE_DAEMON_NEEDED:     clients.DaemonNeeded,
		dlv1.Cause_CAUSE_WRONG_DAEMON:      clients.WrongDaemon,
		dlv1.Cause_CAUSE_NOT_KEPT:          clients.NotKept,
		dlv1.Cause_CAUSE_MOST_MISSING:      clients.MostMissing,
	}
	warningFromWire = map[dlv1.Warning]clients.Warning{
		dlv1.Warning_WARNING_REMOVES_AT_LIMIT:   clients.RemovesAtLimit,
		dlv1.Warning_WARNING_MOVES_AFTER_FINISH: clients.MovesAfterFinish,
		dlv1.Warning_WARNING_NO_MOVING_STATE:    clients.NoMovingState,
	}
)
