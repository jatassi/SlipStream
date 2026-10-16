// Package mail renders and sends every email SlipStream sends, over the
// one SMTP server an Admin sets up. It is the only importer of go-mail.
package mail

import (
	"context"
	"time"
)

// Address is a mailbox and the name shown with it.
type Address struct {
	Name  string
	Email string
}

// Message is a rendered email: its subject, its HTML part and its
// plain-text part.
type Message struct {
	To      Address
	Subject string
	HTML    string
	Text    string
}

type OutcomeKind string

const (
	Sent     OutcomeKind = "sent"
	Failed   OutcomeKind = "failed"
	NotSetUp OutcomeKind = "not-set-up"
)

// Outcome is what became of one message. Reply is the server's reply, or
// why no server answered, for Failed.
type Outcome struct {
	Kind  OutcomeKind
	Reply string
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
