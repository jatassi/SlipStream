package providers

// Gotify pushes a message to an application on the Admin's own Gotify
// server, with the poster as a big image the phone fetches from the
// metadata source's public URL.
//
// Checked on 2026-11-16 against Gotify's official documentation, read from
// its source repositories because gotify.net is unreachable from this
// network:
//   - github.com/gotify/website, docs/pushmsg.md: POST {server}/message with
//     the X-Gotify-Key header, title, message (the only required field)
//     and priority.
//   - docs/priority.md: 0 shows no notification, 4 to 7 add a sound and
//     8 to 10 add vibration on Android.
//   - docs/msgextras.md: extras client::display contentType,
//     client::notification click.url and bigImageUrl.
//   - github.com/gotify/server, docs/spec.json: /message takes
//     application/json, extras are accepted only in JSON, and it answers
//     200, 400, 401 or 403. No length limit or rate limit is documented.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

var gotify = Provider{
	ID:    "gotify",
	Label: "Gotify",
	Fields: []Field{
		{Key: "serverUrl", Label: "Server URL", Kind: URL, Help: "The address of your Gotify server, such as https://gotify.example.com."},
		{Key: "appToken", Label: "Application token", Kind: Text, Secret: true, Help: "Make an application on your Gotify server and copy its token."},
		{Key: "priority", Label: "Priority", Kind: Number, Details: true, Default: json.RawMessage("5"), Min: intp(0), Max: intp(10), Help: "Gotify's Android app plays a sound from 4 and vibrates from 8. At 0 it shows no notification."},
	},
	Image:    PublicURL,
	Host:     enteredHost("serverUrl"),
	Build:    buildGotify,
	Classify: reading{image: true, auth: "appToken", gone: "serverUrl"}.classify,
	Words: map[ReasonCode]string{
		Unauthorized: "Gotify responded that the application token is wrong. Paste the token of an application on your Gotify server.",
		Gone:         "There is no Gotify server at this URL. Check the server URL.",
	},
}

type gotifyNotification struct {
	Click       *gotifyClick `json:"click,omitempty"`
	BigImageURL string       `json:"bigImageUrl,omitempty"`
}

type gotifyClick struct {
	URL string `json:"url"`
}

func buildGotify(p Payload) (Outbound, error) {
	u, err := url.Parse(p.text("serverUrl"))
	if err != nil {
		return Outbound{}, fmt.Errorf("read the Gotify server URL: %w", err)
	}
	n := gotifyNotification{}
	if link := p.Message.Link.Path; link != "" {
		n.Click = &gotifyClick{URL: link}
	}
	if p.Poster != nil {
		n.BigImageURL = p.Poster.URL
	}
	body, err := json.Marshal(struct {
		Title    string         `json:"title"`
		Message  string         `json:"message"`
		Priority int            `json:"priority"`
		Extras   map[string]any `json:"extras"`
	}{
		Title:    p.Message.Title,
		Message:  joinLines(p.Message.Lines),
		Priority: p.number("priority"),
		Extras: map[string]any{
			"client::display":      map[string]string{"contentType": "text/plain"},
			"client::notification": n,
		},
	})
	if err != nil {
		return Outbound{}, fmt.Errorf("encode the Gotify message: %w", err)
	}
	header := jsonHeader()
	header.Set("X-Gotify-Key", p.text("appToken"))
	return Outbound{Post: &Request{Method: http.MethodPost, URL: u.JoinPath("message").String(), Header: header, Body: body}}, nil
}

func intp(n int) *int { return &n }
