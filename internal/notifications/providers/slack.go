package providers

// Slack posts a message to an incoming webhook's channel, with the poster
// as an image Slack fetches from the metadata source's public URL, since
// an incoming webhook can't upload a file.
//
// docs.slack.dev and api.slack.com are unreachable from this network, so
// on 2026-11-16 none of this was checked against Slack's own pages. It
// rests on https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks
// and https://docs.slack.dev/apis/web-api/rate-limits as recalled: a POST
// of {text, blocks} to the webhook URL, a 200 answering "ok", a section's
// mrkdwn text of at most 3000 characters with &, < and > escaped, an
// image accessory's image_url fetched by Slack, a 429 with Retry-After,
// and plain-text errors: 400 invalid_payload, 403 action_prohibited, 404
// no_service or channel_not_found, and 410 channel_is_archived.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const slackSection = 3000

var slack = Provider{
	ID:    "slack",
	Label: "Slack",
	Fields: []Field{
		{Key: "webhookUrl", Label: "Webhook URL", Kind: URL, Secret: true, Shown: ShownHost, Help: "In your Slack app's settings, open Incoming Webhooks and copy the channel's webhook URL."},
	},
	Image:    PublicURL,
	Host:     enteredHost("webhookUrl"),
	Build:    buildSlack,
	Classify: reading{image: true, auth: "webhookUrl", gone: "webhookUrl"}.classify,
	Words: map[ReasonCode]string{
		Gone:         "Slack responded that this webhook or its channel no longer exists. Paste a new webhook URL.",
		Unauthorized: "Slack responded that this webhook can't post to its channel. Paste a new webhook URL.",
	},
}

type slackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type slackImage struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url"`
	AltText  string `json:"alt_text"`
}

type slackSectionBlock struct {
	Type      string      `json:"type"`
	Text      slackText   `json:"text"`
	Accessory *slackImage `json:"accessory,omitempty"`
}

func buildSlack(p Payload) (Outbound, error) {
	title := escapeMrkdwn(p.Message.Title)
	if link := p.Message.Link.Path; link != "" {
		title = "<" + escapeMrkdwn(link) + "|" + title + ">"
	}
	lines := make([]string, len(p.Message.Lines))
	for i, l := range p.Message.Lines {
		lines[i] = escapeMrkdwn(l)
	}
	section := slackSectionBlock{
		Type: "section",
		Text: slackText{Type: "mrkdwn", Text: cut(strings.Join(append([]string{"*" + title + "*"}, lines...), "\n"), slackSection)},
	}
	if p.Poster != nil {
		section.Accessory = &slackImage{Type: "image", ImageURL: p.Poster.URL, AltText: cut(p.Message.Title, slackSection)}
	}
	body, err := json.Marshal(struct {
		Text   string              `json:"text"`
		Blocks []slackSectionBlock `json:"blocks"`
	}{Text: escapeMrkdwn(p.Message.Title), Blocks: []slackSectionBlock{section}})
	if err != nil {
		return Outbound{}, fmt.Errorf("encode the Slack message: %w", err)
	}
	return Outbound{Post: &Request{Method: http.MethodPost, URL: p.text("webhookUrl"), Header: jsonHeader(), Body: body}}, nil
}
