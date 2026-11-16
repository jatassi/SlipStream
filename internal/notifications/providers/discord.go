package providers

// Discord posts an embed to a channel's webhook, with the poster uploaded
// as the embed's thumbnail.
//
// Checked on 2026-11-16 against Discord's official documentation, read from
// its source repository because docs.discord.com is unreachable from this
// network (github.com/discord/discord-api-docs, main):
//   - developers/resources/webhook.mdx, Execute Webhook: the route, ?wait,
//     content, username, avatar_url, embeds (up to 10), files[n],
//     payload_json and attachments.
//   - developers/reference.mdx, Uploading Files: multipart files[n]
//     referenced as attachment://filename, a 20 MiB default per file.
//   - developers/resources/message.mdx, Embed Limits: title 256,
//     description 4096 and 6000 across all embeds, or a 400.
//   - developers/topics/rate-limits.mdx: a 429 with a JSON body whose
//     retry_after is float seconds, and a Retry-After header.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
)

const (
	discordTitle       = 256
	discordDescription = 4096
)

var discordColors = map[notice.Tone]int{
	notice.Info: 0x3B82F6,
	notice.Good: 0x22C55E,
	notice.Warn: 0xF59E0B,
	notice.Fail: 0xEF4444,
}

var discord = Provider{
	ID:    "discord",
	Label: "Discord",
	Fields: []Field{
		{Key: "webhookUrl", Label: "Webhook URL", Kind: URL, Secret: true, Shown: ShownHost, Help: "In Discord, open the channel's settings, then Integrations, then Webhooks, and copy a webhook's URL."},
		{Key: "username", Label: "Username", Kind: Text, Optional: true, Details: true, Help: "The name messages show. Empty uses the webhook's own name."},
		{Key: "avatarUrl", Label: "Avatar URL", Kind: URL, Optional: true, Details: true, Help: "The picture messages show. Empty uses the webhook's own picture."},
	},
	Image:    Bytes,
	Host:     enteredHost("webhookUrl"),
	Build:    buildDiscord,
	Classify: reading{image: true, auth: "webhookUrl", gone: "webhookUrl", own: discordWait}.classify,
	Words: map[ReasonCode]string{
		Gone:         "Discord responded that this webhook no longer exists. Paste a new webhook URL.",
		Unauthorized: "Discord responded that the webhook's token is wrong. Paste the webhook URL again.",
	},
}

type discordMessage struct {
	Username        string            `json:"username,omitempty"`
	AvatarURL       string            `json:"avatar_url,omitempty"`
	Embeds          []discordEmbed    `json:"embeds"`
	AllowedMentions discordMentions   `json:"allowed_mentions"`
	Attachments     []discordAttached `json:"attachments,omitempty"`
}

type discordEmbed struct {
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	URL         string        `json:"url,omitempty"`
	Color       int           `json:"color"`
	Timestamp   time.Time     `json:"timestamp"`
	Thumbnail   *discordImage `json:"thumbnail,omitempty"`
}

type discordImage struct {
	URL string `json:"url"`
}

// discordMentions is empty, so a title holding @everyone pings nobody.
type discordMentions struct {
	Parse []string `json:"parse"`
}

type discordAttached struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
}

func buildDiscord(p Payload) (Outbound, error) {
	u, err := url.Parse(p.text("webhookUrl"))
	if err != nil {
		return Outbound{}, fmt.Errorf("read the Discord webhook URL: %w", err)
	}
	q := u.Query()
	q.Set("wait", "true")
	u.RawQuery = q.Encode()
	m := discordMessage{
		Username:  p.text("username"),
		AvatarURL: p.text("avatarUrl"),
		Embeds: []discordEmbed{{
			Title:       fitEscaped(p.Message.Title, discordTitle, escapeMarkdown),
			Description: fitEscaped(joinLines(p.Message.Lines), discordDescription, escapeMarkdown),
			URL:         p.Message.Link.Path,
			Color:       discordColors[p.Message.Tone],
			Timestamp:   p.At.UTC(),
		}},
		AllowedMentions: discordMentions{Parse: []string{}},
	}
	if p.Poster == nil {
		body, err := json.Marshal(m)
		if err != nil {
			return Outbound{}, fmt.Errorf("encode the Discord message: %w", err)
		}
		return Outbound{Post: &Request{Method: http.MethodPost, URL: u.String(), Header: jsonHeader(), Body: body}}, nil
	}
	m.Embeds[0].Thumbnail = &discordImage{URL: "attachment://" + p.Poster.Name}
	m.Attachments = []discordAttached{{ID: 0, Filename: p.Poster.Name}}
	payload, err := json.Marshal(m)
	if err != nil {
		return Outbound{}, fmt.Errorf("encode the Discord message: %w", err)
	}
	body, ctype := multipartBody([]part{{name: "payload_json", body: payload}, posterPart("files[0]", p.Poster)})
	return Outbound{Post: &Request{Method: http.MethodPost, URL: u.String(), Header: http.Header{"Content-Type": {ctype}}, Body: body}}, nil
}

// discordWait reads a 429's retry_after, float seconds, which is finer
// than the Retry-After header's whole seconds.
func discordWait(r Response, now time.Time) (Verdict, bool) {
	if r.Status != http.StatusTooManyRequests {
		return Verdict{}, false
	}
	var limited struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if json.Unmarshal(r.Body, &limited) != nil || limited.RetryAfter <= 0 {
		return Verdict{}, false
	}
	return slowDown(now, time.Duration(limited.RetryAfter*float64(time.Second)), r.Status), true
}
