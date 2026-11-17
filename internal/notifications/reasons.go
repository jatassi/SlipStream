package notifications

import (
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

// The codes of what statusOf derives from the roster and of a seal that
// won't open. The first two are never stored; the third is stored on a
// paused line.
const (
	nodeRemoved       providers.ReasonCode = "node_removed"
	nodeOffline       providers.ReasonCode = "node_offline"
	secretsUnreadable providers.ReasonCode = "secrets_unreadable"
)

// wording words a stored reason when it is read, so a copy change reaches
// a line paused long ago.
type wording struct {
	// names is a Node's name now, "a removed Node" once it is gone.
	names  func(role.NodeID) string
	now    time.Time
	zone   *time.Location
	giveUp time.Duration
}

// reason is p's own sentence for r's code when p words it, else the
// generic one.
func (w wording) reason(p providers.Provider, r providers.Reason) string {
	if s, ok := p.Words[r.Code]; ok {
		return s
	}
	node := w.names(r.Node)
	switch r.Code {
	case providers.Gone:
		return p.Label + " responded that this address no longer exists. Edit the connection with a new one."
	case providers.Unauthorized:
		return p.Label + " responded that it doesn't accept these credentials. Edit the connection with new ones."
	case providers.ChatNotFound:
		return p.Label + " responded that the chat doesn't exist. Check the chat id."
	case providers.UserUnknown:
		return p.Label + " responded that the user key doesn't exist. Check the user key."
	case providers.BadRequest:
		return fmt.Sprintf("%s responded with HTTP %d. Check the connection's settings.", p.Label, r.Status)
	case providers.TooLarge:
		return p.Label + " responded that the message is too large."
	case providers.Refused:
		return fmt.Sprintf("%s responded with HTTP %d to this message.", p.Label, r.Status)
	case providers.RateLimited:
		return p.Label + " responded with a limit. Messages wait until " + w.when(r.Until) + "."
	case providers.ServerError:
		return fmt.Sprintf("%s responded with HTTP %d. SlipStream retries automatically.", p.Label, r.Status)
	case providers.Network:
		return p.Label + " didn't respond. SlipStream retries automatically."
	case providers.NodeSilent:
		return node + ", which sends notifications, didn't respond. SlipStream retries automatically."
	case providers.NoNode:
		return "No Node with the Notifying Role can send now. SlipStream retries automatically."
	case providers.Unreadable:
		return node + "'s response couldn't be read. SlipStream retries automatically."
	case providers.ScriptExit:
		if tail := strings.TrimSpace(r.Tail); tail != "" {
			return fmt.Sprintf("The script on %s exited with code %d: %s", node, r.Exit, tail)
		}
		return fmt.Sprintf("The script on %s exited with code %d. SlipStream retries automatically.", node, r.Exit)
	case providers.ScriptStopped:
		return "The script on " + node + " ran longer than 'Stop the script after' allows and was stopped. SlipStream retries automatically."
	case providers.ScriptMissing:
		return node + " has no file at the script's path. Fix the path or pick another Node."
	case providers.MailNotSetUp:
		return "Email isn't set up. Set it up in the Email settings."
	case providers.MailAuth:
		return "The mail server didn't accept SlipStream's sign-in. Check the Email settings."
	case providers.MailRecipient:
		return "The mail server responded that it can't deliver to " + r.To + ". Check the address."
	case providers.MailData:
		return "The mail server responded that it can't take this message."
	case providers.TooOld:
		return "It waited longer than " + setting.Words(w.giveUp) + "."
	case nodeRemoved:
		return node + " was removed. Pick another Node for this script."
	case nodeOffline:
		return node + " runs this script, and " + node + " is offline. The messages wait until it is back."
	case secretsUnreadable:
		return "SlipStream can't open this connection's saved secrets. Enter them again."
	}
	return p.Label + " didn't take the message. SlipStream retries automatically."
}

// when is t as a time today, else as a date.
func (w wording) when(t time.Time) string {
	t, now := t.In(w.zone), w.now.In(w.zone)
	if t.YearDay() == now.YearDay() && t.Year() == now.Year() {
		return t.Format("15:04")
	}
	return t.Format("2 January")
}
