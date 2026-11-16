package providers

// Pushbullet pushes a note, or a link when the message has one, to every
// device of the account or to one device. It takes no poster, since an
// image needs a three-step upload for a push that shows only its name.
//
// docs.pushbullet.com is unreachable from this network, so on 2026-11-16
// none of this was checked against Pushbullet's own page. It rests on
// https://docs.pushbullet.com/#create-push as recalled: POST /v2/pushes
// with the Access-Token header and JSON {type: note or link, title, body,
// url, device_iden}; errors as {"error": {"code", "type", "message"}} with
// 401 for a wrong token and 403 for a Pro feature; and a 429 past the
// limit, with X-Ratelimit-Reset as Unix seconds.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var pushbullet = Provider{
	ID:    "pushbullet",
	Label: "Pushbullet",
	Fields: []Field{
		{Key: "accessToken", Label: "Access token", Kind: Text, Secret: true, Help: "Make one under Settings, then Account, on pushbullet.com."},
		{Key: "device", Label: "Device", Kind: Text, Optional: true, Details: true, Help: "One device's iden. Empty sends to every device."},
	},
	Image:    NoImage,
	Host:     fixedHost(func(e Endpoints) endpoint.URL { return e.Pushbullet }),
	Build:    buildPushbullet,
	Classify: reading{auth: "accessToken", gone: "device", own: pushbulletWait}.classify,
	Words: map[ReasonCode]string{
		Unauthorized: "Pushbullet responded that the access token is wrong. Make a new one under Settings, then Account, on pushbullet.com.",
	},
}

func buildPushbullet(p Payload) (Outbound, error) {
	push := struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		URL    string `json:"url,omitempty"`
		Device string `json:"device_iden,omitempty"`
	}{Type: "note", Title: p.Message.Title, Body: joinLines(p.Message.Lines), Device: p.text("device")}
	if link := p.Message.Link.Path; link != "" {
		push.Type, push.URL = "link", link
	}
	body, err := json.Marshal(push)
	if err != nil {
		return Outbound{}, fmt.Errorf("encode the Pushbullet push: %w", err)
	}
	header := jsonHeader()
	header.Set("Access-Token", p.text("accessToken"))
	return Outbound{Post: &Request{Method: http.MethodPost, URL: p.Endpoints.Pushbullet.JoinPath("v2", "pushes").String(), Header: header, Body: body}}, nil
}

// pushbulletWait waits a 429 out until X-Ratelimit-Reset.
func pushbulletWait(r Response, now time.Time) (Verdict, bool) {
	if r.Status != http.StatusTooManyRequests {
		return Verdict{}, false
	}
	reset := epoch(r.Header.Get("X-Ratelimit-Reset"))
	if !reset.After(now) {
		return Verdict{}, false
	}
	return slowDown(now, reset.Sub(now), r.Status), true
}
