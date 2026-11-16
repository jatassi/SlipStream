package providers

// Webhook sends the Webhook body, version 1, to a URL the Admin owns, with
// the poster as a public URL in the body. SlipStream defines the body, so
// this file follows no provider's page.

import (
	"encoding/json"
	"net/http"
)

var webhook = Provider{
	ID:    "webhook",
	Label: "Webhook",
	Fields: []Field{
		{Key: "url", Label: "URL", Kind: URL, Secret: true, Shown: ShownHost, Help: "SlipStream sends each message to this URL as JSON."},
		{Key: "method", Label: "Method", Kind: Select, Details: true, Default: json.RawMessage(`"POST"`), Options: []Option{
			{Value: http.MethodPost, Label: "POST"}, {Value: http.MethodPut, Label: "PUT"},
		}},
		{Key: "username", Label: "Username", Kind: Text, Optional: true, Details: true, Help: "Sent with the password as Basic authentication."},
		{Key: "password", Label: "Password", Kind: Text, Secret: true, Optional: true, Details: true},
		{Key: "headers", Label: "Headers", Kind: Headers, Secret: true, Optional: true, Details: true, Help: "Sent with each message, such as a token your server checks."},
	},
	Image:    BodyURL,
	Host:     enteredHost("url"),
	Build:    buildWebhook,
	Classify: reading{image: true, auth: "password", gone: "url"}.classify,
	Words: map[ReasonCode]string{
		Gone:         "The webhook's server responded that there is nothing at this URL. Check the URL.",
		Unauthorized: "The webhook's server responded that SlipStream isn't signed in. Check the username, password and headers.",
	},
}

func buildWebhook(p Payload) (Outbound, error) {
	header := jsonHeader()
	if user := p.text("username"); user != "" {
		header.Set("Authorization", basicAuth(user, p.text("password")))
	}
	for _, h := range p.headers("headers") {
		header.Set(h.Name, h.Value)
	}
	return Outbound{Post: &Request{Method: p.text("method"), URL: p.text("url"), Header: header, Body: bodyOf(p)}}, nil
}
