package providers

// Apprise sends through the Admin's Apprise API server, to the services a
// saved configuration key's tags select or to a list of Apprise URLs, with
// the poster as an attachment.
//
// Checked on 2026-11-16 against Apprise API's official README
// (github.com/caronc/apprise-api, master, README.md):
//   - POST /notify/ with urls, and POST /notify/{KEY} with tag (a comma is
//     OR, a space is AND); body (required), title, type (info, success,
//     warning or failure) and format (text, markdown or html); JSON, a form,
//     or multipart with the file in attach.
//   - API Response Codes: 200; 400 for an invalid key, payload, field,
//     format or tag; 401 for missing or wrong Basic Auth; 404 for a key
//     with no saved configuration; 413 for a body over the upload limit;
//     424 when at least one notification failed or no tag matched; and 429
//     from its Nginx with Retry-After: 60.
//   - APPRISE_ATTACH_SIZE: attachments up to 200 MB by default, and none
//     at 0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
)

var apprise = Provider{
	ID:    "apprise",
	Label: "Apprise",
	Fields: []Field{
		{Key: "serverUrl", Label: "Server URL", Kind: URL, Help: "The address of your Apprise API server, with its port."},
		{Key: "key", Label: "Configuration key", Kind: Text, Secret: true, Optional: true, Help: "The key of a configuration saved on the server. Leave it empty to list Apprise URLs instead."},
		{Key: "urls", Label: "Apprise URLs", Kind: List, Secret: true, Optional: true, Help: "The services to send to, each an Apprise URL such as a join:// or mailto:// one. Used when no key is set."},
		{Key: "tags", Label: "Tags", Kind: List, Optional: true, Details: true, Help: "With a key, sends only to the services that have one of these tags."},
		{Key: "username", Label: "Username", Kind: Text, Optional: true, Details: true, Help: "Needed only when the server asks for a sign-in."},
		{Key: "password", Label: "Password", Kind: Text, Secret: true, Optional: true, Details: true},
	},
	Image:    Bytes,
	Host:     enteredHost("serverUrl"),
	Build:    buildApprise,
	Classify: reading{image: true, auth: "password", gone: "key", own: appriseAnswer}.classify,
	Words: map[ReasonCode]string{
		BadRequest:   "Apprise responded that the key, tags or Apprise URLs can't be used. Check them.",
		Gone:         "Apprise responded that it has no configuration saved with this key. Check the key.",
		Unauthorized: "Apprise responded that the username or password is wrong.",
	},
}

var appriseTypes = map[notice.Tone]string{
	notice.Info: "info",
	notice.Good: "success",
	notice.Warn: "warning",
	notice.Fail: "failure",
}

func buildApprise(p Payload) (Outbound, error) {
	server, err := url.Parse(p.text("serverUrl"))
	if err != nil {
		return Outbound{}, fmt.Errorf("read the Apprise server URL: %w", err)
	}
	text := joinLines(p.Message.Lines)
	if link := p.Message.Link.Path; link != "" {
		text += "\n" + link
	}
	fields := [][2]string{{"title", p.Message.Title}, {"body", text}, {"type", appriseTypes[p.Message.Tone]}, {"format", "text"}}
	u := server.JoinPath("notify")
	if key := p.text("key"); key != "" {
		u = server.JoinPath("notify", key)
		if tags := p.list("tags"); len(tags) > 0 {
			fields = append(fields, [2]string{"tag", strings.Join(tags, ",")})
		}
	} else {
		u.Path += "/"
		fields = append(fields, [2]string{"urls", strings.Join(p.list("urls"), ",")})
	}
	header := http.Header{}
	if user := p.text("username"); user != "" {
		header.Set("Authorization", basicAuth(user, p.text("password")))
	}
	var body []byte
	if p.Poster == nil {
		obj := make(map[string]string, len(fields))
		for _, f := range fields {
			obj[f[0]] = f[1]
		}
		if body, err = json.Marshal(obj); err != nil {
			return Outbound{}, fmt.Errorf("encode the Apprise message: %w", err)
		}
		header.Set("Content-Type", "application/json")
	} else {
		parts := make([]part, 0, len(fields)+1)
		for _, f := range fields {
			parts = append(parts, part{name: f[0], body: []byte(f[1])})
		}
		var ctype string
		body, ctype = multipartBody(append(parts, posterPart("attach", p.Poster)))
		header.Set("Content-Type", ctype)
	}
	return Outbound{Post: &Request{Method: http.MethodPost, URL: u.String(), Header: header, Body: body}}, nil
}

// appriseAnswer reads a 424 as sent, since some of the services may have
// taken it and a resend would repeat it to them, and a 400 as a refusal of
// the connection's key, tags or URLs.
func appriseAnswer(r Response, _ time.Time) (Verdict, bool) {
	switch r.Status {
	case http.StatusFailedDependency:
		return Verdict{Outcome: Sent, Response: r.Body}, true
	case http.StatusBadRequest:
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: BadRequest, Status: r.Status, Field: "key"}, Response: r.Body}, true
	}
	return Verdict{}, false
}
