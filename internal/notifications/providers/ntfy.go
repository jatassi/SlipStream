package providers

// ntfy publishes a message to a topic on ntfy.sh or the Admin's own
// server, with the poster uploaded as the message's attachment.
//
// Checked on 2026-11-16 against ntfy's official documentation, read from
// its source repository because docs.ntfy.sh is unreachable from this
// network (github.com/binwiederhier/ntfy, main):
//   - docs/publish.md: POST or PUT {server}/{topic} with the message as the
//     body; the headers Title, Priority (1 to 5), Click, Message and
//     Filename; a file PUT as the body with Filename and the text in the
//     Message header; any header RFC 2047 encoded when it isn't ASCII;
//     Authorization: Bearer with an access token.
//   - docs/publish.md, Limitations: a message of up to 4,096 bytes (a
//     longer one becomes an attachment), a title of 1 KB, and attachments
//     of 15 MB by default but 2 MB on ntfy.sh.
//   - server/errors.go: 401 and 403 for authentication, 413 for an
//     attachment too large, and 429 limits, sent with Retry-After.

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
)

const (
	ntfyMessage = 4096
	ntfyTitle   = 1024
)

var ntfy = Provider{
	ID:    "ntfy",
	Label: "ntfy",
	Fields: []Field{
		{Key: "topic", Label: "Topic", Kind: Text, Secret: true, Help: "Anyone who knows the topic can read it, so pick a name that's hard to guess."},
		{Key: "serverUrl", Label: "Server URL", Kind: URL, Optional: true, Details: true, Help: "Your own ntfy server. Empty sends to ntfy.sh."},
		{Key: "accessToken", Label: "Access token", Kind: Text, Secret: true, Optional: true, Details: true, Help: "Needed only when the topic is protected."},
		{Key: "priority", Label: "Priority", Kind: Select, Details: true, Default: json.RawMessage(`"3"`), Options: []Option{
			{Value: "1", Label: "Min"}, {Value: "2", Label: "Low"}, {Value: "3", Label: "Default"}, {Value: "4", Label: "High"}, {Value: "5", Label: "Max"},
		}},
	},
	Image:    Bytes,
	Host:     ntfyHost,
	Build:    buildNtfy,
	Classify: reading{image: true, auth: "accessToken", gone: "serverUrl"}.classify,
	Words: map[ReasonCode]string{
		Unauthorized: "ntfy responded that this topic needs an access token, or that the token is wrong. Check the access token.",
		Gone:         "There is no ntfy server at this URL. Check the server URL.",
	},
}

// ntfyServer is the server URL the Admin entered, or ntfy.sh's endpoint.
func ntfyServer(server string, e Endpoints) (*url.URL, error) {
	if server == "" {
		return e.Ntfy.JoinPath(), nil
	}
	u, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("read the ntfy server URL: %w", err)
	}
	return u, nil
}

func ntfyHost(values Values, e Endpoints) (*url.URL, bool) {
	u, err := ntfyServer(decode[string](values["serverUrl"]), e)
	if err != nil || u.Host == "" {
		return nil, false
	}
	return &url.URL{Scheme: u.Scheme, Host: u.Host}, true
}

func buildNtfy(p Payload) (Outbound, error) {
	server, err := ntfyServer(p.text("serverUrl"), p.Endpoints)
	if err != nil {
		return Outbound{}, err
	}
	header := http.Header{}
	header.Set("Title", mime.BEncoding.Encode("UTF-8", cutBytes(p.Message.Title, ntfyTitle)))
	header.Set("Priority", p.text("priority"))
	if link := p.Message.Link.Path; link != "" {
		header.Set("Click", link)
	}
	if token := p.text("accessToken"); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	message := cutBytes(joinLines(p.Message.Lines), ntfyMessage)
	u := server.JoinPath(p.text("topic")).String()
	if p.Poster == nil {
		header.Set("Content-Type", "text/plain; charset=utf-8")
		return Outbound{Post: &Request{Method: http.MethodPost, URL: u, Header: header, Body: []byte(message)}}, nil
	}
	header.Set("Content-Type", p.Poster.Type)
	header.Set("Filename", p.Poster.Name)
	header.Set("Message", mime.BEncoding.Encode("UTF-8", message))
	return Outbound{Post: &Request{Method: http.MethodPut, URL: u, Header: header, Body: p.Poster.Bytes}}, nil
}
