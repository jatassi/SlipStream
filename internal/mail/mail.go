// Package mail renders and sends every email SlipStream sends, over the
// one SMTP server an Admin sets up. It is the only importer of go-mail.
package mail

import (
	"context"
	"time"
)

type Address struct {
	Name  string
	Email string
}

type Message struct {
	To      Address
	Subject string
	HTML    string
	Text    string
	// Inline are the parts HTML shows through cid:<ContentID>.
	Inline []Part
}

// Part is one file sent inside a message, such as a poster.
type Part struct {
	ContentID string
	Type      string
	Name      string
	Body      []byte
}

type OutcomeKind string

const (
	Sent     OutcomeKind = "sent"
	Failed   OutcomeKind = "failed"
	NotSetUp OutcomeKind = "not-set-up"
)

// Step is where in a send the server answered.
type Step string

const (
	// StepConnect is the dial, TLS and AUTH.
	StepConnect Step = "connect"
	StepFrom    Step = "from"
	StepRcpt    Step = "rcpt"
	StepData    Step = "data"
)

// Outcome is what became of one message. Reply is the server's reply, or
// why no server answered, for Failed. Permanent is a 5xx reply, which
// sending again won't change, and Step is where a Failed send stopped.
type Outcome struct {
	Kind      OutcomeKind
	Reply     string
	Permanent bool
	Step      Step
}

// Sender sends one message and waits for its outcome. A mail server's
// refusal is an Outcome; the error is SlipStream's own failure. Send
// panics inside a storage Write, since a send holds the writer for as long
// as the server takes.
type Sender interface {
	Send(ctx context.Context, m Message) (Outcome, error)
}

type Timings struct {
	// Send bounds one message: the dial, TLS, AUTH and the data.
	Send time.Duration `yaml:"send"`
}

func DefaultTimings() Timings { return Timings{Send: 15 * time.Second} }
