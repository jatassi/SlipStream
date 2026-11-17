package providers

import (
	"time"

	"github.com/jatassi/slipstream/internal/role"
)

// ReasonCode names why a send didn't go. It is stored, so a code is never
// renamed.
type ReasonCode string

const (
	// Gone: 404 or 410.
	Gone ReasonCode = "gone"
	// Unauthorized: 401 or 403.
	Unauthorized ReasonCode = "unauthorized"
	// ChatNotFound: Telegram's chat not found.
	ChatNotFound ReasonCode = "chat_not_found"
	// UserUnknown: Pushover's unknown user.
	UserUnknown ReasonCode = "user_unknown"
	// BadRequest: an Apprise 400, a refusal of the connection.
	BadRequest ReasonCode = "bad_request"
	// TooLarge: 413.
	TooLarge ReasonCode = "too_large"
	// Refused: another 400 or 422 refusing the content.
	Refused     ReasonCode = "refused"
	RateLimited ReasonCode = "rate_limited"
	// ServerError: a 5xx, or another status.
	ServerError ReasonCode = "server_error"
	// Network: no response from the provider.
	Network ReasonCode = "network"
	// NodeSilent: the Node didn't respond.
	NodeSilent ReasonCode = "node_silent"
	// NoNode: no Node has the Notifying Role.
	NoNode ReasonCode = "no_node"
	// Unreadable: the Node's answer broke a cap.
	Unreadable    ReasonCode = "unreadable"
	ScriptExit    ReasonCode = "script_exit"
	ScriptStopped ReasonCode = "script_stopped"
	ScriptMissing ReasonCode = "script_missing"
	MailNotSetUp  ReasonCode = "mail_not_set_up"
	MailAuth      ReasonCode = "mail_auth"
	MailRecipient ReasonCode = "mail_recipient"
	MailData      ReasonCode = "mail_data"
	// TooOld: given up after "Give up on a message after".
	TooOld ReasonCode = "too_old"
)

// Reason is a failure as facts, stored as JSON and worded when read. Fields
// are only ever added.
type Reason struct {
	Code   ReasonCode `json:"code"`
	Status int        `json:"status,omitempty"`
	// Field is the field the Admin should change.
	Field string      `json:"field,omitempty"`
	Node  role.NodeID `json:"node,omitzero"`
	// Until is when a rate_limited wait ends.
	Until time.Time `json:"until,omitzero"`
	// Exit is a script_exit's code.
	Exit int `json:"exit,omitempty"`
	// Tail is at most 1 KiB of a script's output, for script_exit and
	// script_stopped.
	Tail string `json:"tail,omitempty"`
	// To is a mail_recipient's address.
	To string `json:"to,omitempty"`
}
