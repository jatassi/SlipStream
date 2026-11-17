package providers

// Pushover sends a push to a user's or group's devices, with the poster as
// an attachment.
//
// pushover.net is unreachable from this network, so on 2026-11-16 none of
// this was checked against Pushover's own page. It rests on
// https://pushover.net/api as recalled: POST /1/messages.json as a form, or
// multipart with an attachment; token, user, message, title, url,
// url_title, priority -2 to 2 (2 needs retry and expire, so it isn't
// offered), sound and device; a message of 1024 characters, a title of
// 250, a url of 512 and a url_title of 100; an answer of {"status": 1} or a
// 4xx with {"status": 0, "errors": [...]} naming the bad field as
// "user": "invalid" or "token": "invalid"; and a 429 once the monthly limit
// is spent, with X-Limit-App-Reset as Unix seconds. The attachment limit
// is 2.5 MB in older copies of the page and 5 MB in newer ones.

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const (
	pushoverMessage  = 1024
	pushoverTitle    = 250
	pushoverURL      = 512
	pushoverURLTitle = 100
)

var pushover = Provider{
	ID:    "pushover",
	Label: "Pushover",
	Fields: []Field{
		{Key: "appToken", Label: "Application token", Kind: Text, Secret: true, Help: "Make an application on pushover.net and copy its API token."},
		{Key: "userKey", Label: "User key", Kind: Text, Secret: true, Help: "Your user key, or a group's key, from your Pushover dashboard."},
		{Key: "devices", Label: "Devices", Kind: List, Optional: true, Details: true, Help: "The device names to send to. Empty sends to every device."},
		{Key: "priority", Label: "Priority", Kind: Select, Details: true, Default: json.RawMessage(`"0"`), Options: []Option{
			{Value: "-2", Label: "Lowest"}, {Value: "-1", Label: "Low"}, {Value: "0", Label: "Normal"}, {Value: "1", Label: "High"},
		}},
		{Key: "sound", Label: "Sound", Kind: Text, Optional: true, Details: true, Help: "The name of a Pushover sound, such as pushover or none. Empty uses your default."},
	},
	Image:    Bytes,
	Host:     fixedHost(func(e Endpoints) endpoint.URL { return e.Pushover }),
	Build:    buildPushover,
	Classify: reading{image: true, auth: "appToken", gone: "appToken", own: pushoverAnswer}.classify,
	Words: map[ReasonCode]string{
		UserUnknown:  "Pushover responded that the user key isn't a user or group. Paste your user key from your Pushover dashboard.",
		Unauthorized: "Pushover responded that the application token is wrong. Paste your application's API token.",
	},
}

func buildPushover(p Payload) (Outbound, error) {
	fields := [][2]string{
		{"token", p.text("appToken")},
		{"user", p.text("userKey")},
		{"title", cut(p.Message.Title, pushoverTitle)},
		{"message", cut(joinLines(p.Message.Lines), pushoverMessage)},
		{"priority", p.text("priority")},
		{"timestamp", strconv.FormatInt(p.At.Unix(), 10)},
	}
	if link := p.Message.Link.Path; link != "" && len(link) <= pushoverURL {
		fields = append(fields, [2]string{"url", link}, [2]string{"url_title", cut(linkLabel, pushoverURLTitle)})
	}
	if devices := p.list("devices"); len(devices) > 0 {
		fields = append(fields, [2]string{"device", strings.Join(devices, ",")})
	}
	if sound := p.text("sound"); sound != "" {
		fields = append(fields, [2]string{"sound", sound})
	}
	u := p.Endpoints.Pushover.JoinPath("1", "messages.json").String()
	if p.Poster == nil {
		form := url.Values{}
		for _, f := range fields {
			form.Set(f[0], f[1])
		}
		return Outbound{Post: &Request{Method: http.MethodPost, URL: u, Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, Body: []byte(form.Encode())}}, nil
	}
	parts := make([]part, 0, len(fields)+1)
	for _, f := range fields {
		parts = append(parts, part{name: f[0], body: []byte(f[1])})
	}
	body, ctype := multipartBody(append(parts, posterPart("attachment", p.Poster)))
	return Outbound{Post: &Request{Method: http.MethodPost, URL: u, Header: http.Header{"Content-Type": {ctype}}, Body: body}}, nil
}

// pushoverAnswer reads which key a 4xx names, and when a 429's monthly
// limit resets.
func pushoverAnswer(r Response, now time.Time) (Verdict, bool) {
	if r.Status == http.StatusTooManyRequests {
		if reset := epoch(r.Header.Get("X-Limit-App-Reset")); reset.After(now) {
			return slowDown(now, reset.Sub(now), r.Status), true
		}
		return Verdict{}, false
	}
	if r.Status < 400 || r.Status >= 500 {
		return Verdict{}, false
	}
	var a struct {
		User  string `json:"user"`
		Token string `json:"token"`
	}
	if json.Unmarshal(r.Body, &a) != nil {
		return Verdict{}, false
	}
	switch {
	case a.Token == "invalid":
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: Unauthorized, Status: r.Status, Field: "appToken"}}, true
	case a.User == "invalid":
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: UserUnknown, Status: r.Status, Field: "userKey"}}, true
	}
	return Verdict{}, false
}
