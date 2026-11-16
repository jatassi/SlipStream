package providers

import (
	"net/http"
	"time"
)

// Request is one HTTP request a builder makes. The dispatcher adds the
// timeout and the dial rule when it hands it to a Node.
type Request struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

// Response is what came back, as the Controller checked it.
type Response struct {
	Status int
	// Header holds the retry headers the Node passes.
	Header http.Header
	// Body is at most 4 KiB.
	Body []byte
}

// Script is one run of a Custom Script.
type Script struct {
	Path      string
	Args      []string
	Event     string
	EventFile []byte
	Stop      time.Duration
}

// Ran is how a Script ended.
type Ran struct {
	Exit int
	// Tail is at most the last 1 KiB of its output.
	Tail     []byte
	TimedOut bool
	Missing  bool
}

// Mail is one email to one address.
type Mail struct {
	To      string
	Subject string
	HTML    string
	Text    string
	Inline  []Part
}

// Part is an inline image, referenced from HTML by ContentID.
type Part struct {
	ContentID string
	Type      string
	Name      string
	Body      []byte
}

// MailStep is where an SMTP session stopped.
type MailStep string

const (
	StepConnect MailStep = "connect"
	StepFrom    MailStep = "from"
	StepRcpt    MailStep = "rcpt"
	StepData    MailStep = "data"
)

// MailReply is how one Mail went.
type MailReply struct {
	To        string
	Sent      bool
	NotSetUp  bool
	Permanent bool
	Step      MailStep
}

// Outbound is exactly one of its fields.
type Outbound struct {
	Post *Request
	Run  *Script
	Mail []Mail
}

// Reply is exactly one of Post, Run or Mail, or Down when no answer came.
type Reply struct {
	Post *Response
	Run  *Ran
	Mail []MailReply
	// Down is node_silent, no_node or unreadable, and zero otherwise.
	Down Reason
}
