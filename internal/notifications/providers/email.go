package providers

// Email sends one message to each address through the SMTP server set up
// under Email, with the poster inline. The mail protocol itself is the
// Node's, in internal/mail, so this file follows no provider's page.

import (
	"slices"
	"strings"
	"time"
)

// posterCID is the Content-ID the HTML shows the inline poster by.
const posterCID = "poster"

var email = Provider{
	ID:    "email",
	Label: "Email",
	Fields: []Field{
		{Key: "addresses", Label: "Addresses", Kind: List, Help: "Each address gets its own message, sent through the mail server set up under Email."},
	},
	Image:    Bytes,
	Host:     noHost,
	Build:    buildEmail,
	Classify: classifyEmail,
}

func buildEmail(p Payload) (Outbound, error) {
	text, html := emailBodies(p)
	var out []Mail
	for _, to := range p.list("addresses") {
		if slices.Contains(p.Accepted, to) {
			continue
		}
		m := Mail{To: to, Subject: p.Message.Title, Text: text, HTML: html}
		if p.Poster != nil {
			m.Inline = []Part{{ContentID: posterCID, Type: p.Poster.Type, Name: p.Poster.Name, Body: p.Poster.Bytes}}
		}
		out = append(out, m)
	}
	return Outbound{Mail: out}, nil
}

func emailBodies(p Payload) (string, string) {
	var text, html strings.Builder
	html.WriteString("<h2>" + escapeHTML(p.Message.Title) + "</h2>\n")
	for _, l := range p.Message.Lines {
		text.WriteString(l + "\n")
		html.WriteString("<p>" + escapeHTML(l) + "</p>\n")
	}
	if link := p.Message.Link.Path; link != "" {
		text.WriteString("\n" + linkLabel + ": " + link + "\n")
		html.WriteString(`<p><a href="` + escapeHTML(link) + `">` + linkLabel + "</a></p>\n")
	}
	if p.Poster != nil {
		html.WriteString(`<p><img src="cid:` + posterCID + `" alt="" width="200"></p>` + "\n")
	}
	return text.String(), html.String()
}

// classifyEmail reads each address's reply. A refusal of the mail server
// or of an address pauses the connection, since only the Admin can fix
// it. A refusal of the message gives it up. Anything else is retried,
// to the addresses that didn't take it yet.
func classifyEmail(r Reply, _ time.Time) Verdict {
	if r.Mail == nil {
		return Verdict{Outcome: Transient, Reason: r.Down}
	}
	var accepted []string
	for _, m := range r.Mail {
		if m.Sent {
			accepted = append(accepted, m.To)
		}
	}
	for _, m := range r.Mail {
		switch {
		case m.NotSetUp:
			return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: MailNotSetUp}, Accepted: accepted}
		case m.Permanent && (m.Step == StepConnect || m.Step == StepFrom):
			return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: MailAuth}, Accepted: accepted}
		case m.Permanent && m.Step == StepRcpt:
			return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: MailRecipient, To: m.To, Field: "addresses"}, Accepted: accepted}
		case m.Permanent && m.Step == StepData:
			return Verdict{Outcome: RefusedContent, Reason: Reason{Code: MailData}, Accepted: accepted}
		}
	}
	if len(accepted) == len(r.Mail) {
		return Verdict{Outcome: Sent, Accepted: accepted}
	}
	return Verdict{Outcome: Transient, Reason: Reason{Code: Network}, Accepted: accepted}
}
