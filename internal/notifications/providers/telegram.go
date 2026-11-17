package providers

// Telegram sends a message to a chat or a topic through a bot, or the
// poster as a photo with the message as its caption.
//
// core.telegram.org is unreachable from this network, so on 2026-11-16 the
// Bot API's answers were checked against the source of Telegram's own Bot
// API server (github.com/tdlib/telegram-bot-api, telegram-bot-api/Query.cpp
// and Client.cpp): an error is {"ok": false, "error_code", "description",
// "parameters"} with the HTTP status equal to error_code, a flood wait is
// 429 "Too Many Requests: retry after N" with parameters.retry_after, an
// unknown chat is 400 "Bad Request: chat not found", and a wrong token is
// 401. These come from https://core.telegram.org/bots/api (sendMessage,
// sendPhoto) and were not checked against that page: the routes, chat_id,
// parse_mode HTML with &lt; &gt; &amp; &quot;, message_thread_id,
// disable_notification, link_preview_options, a text of 1 to 4096
// characters and a caption of up to 1024, both counted after entities are
// parsed, and a photo upload of up to 10 MB.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const (
	telegramText    = 4096
	telegramCaption = 1024
	// telegramTitle keeps a long title from taking the lines' room.
	telegramTitle = 256
	linkLabel     = "Open in SlipStream"
)

var telegram = Provider{
	ID:    "telegram",
	Label: "Telegram",
	Fields: []Field{
		{Key: "botToken", Label: "Bot token", Kind: Text, Secret: true, Help: "The token BotFather gave you when you made the bot."},
		{Key: "chatId", Label: "Chat id", Kind: Text, Help: "The chat's id, such as -1001234567890, or a channel's @name. Add the bot to the chat first."},
		{Key: "topicId", Label: "Topic id", Kind: Number, Optional: true, Details: true, Help: "Posts to one topic of a group with topics. Empty posts to the general topic."},
		{Key: "silent", Label: "Send silently", Kind: Switch, Details: true, Default: json.RawMessage("false"), Help: "Members get the message with no sound."},
	},
	Image:    Bytes,
	Host:     fixedHost(func(e Endpoints) endpoint.URL { return e.Telegram }),
	Build:    buildTelegram,
	Classify: reading{image: true, auth: "botToken", gone: "botToken", own: telegramAnswer}.classify,
	Words: map[ReasonCode]string{
		ChatNotFound: "Telegram responded that the chat doesn't exist or the bot isn't in it. Check the chat id, and add the bot to the chat.",
		Unauthorized: "Telegram responded that the bot can't send to this chat. Check the bot token, and that the bot is still in the chat.",
		Gone:         "Telegram responded that there is no bot with this token. Paste the token BotFather gave you.",
	},
}

func buildTelegram(p Payload) (Outbound, error) {
	limit, method := telegramText, "sendMessage"
	if p.Poster != nil {
		limit, method = telegramCaption, "sendPhoto"
	}
	u := p.Endpoints.Telegram.JoinPath("bot"+p.text("botToken"), method).String()
	text := telegramHTML(p.Message.Title, p.Message.Lines, p.Message.Link.Path, limit)
	topic := p.number("topicId")
	if p.Poster == nil {
		m := struct {
			ChatID      string `json:"chat_id"`
			Topic       int    `json:"message_thread_id,omitempty"`
			Text        string `json:"text"`
			ParseMode   string `json:"parse_mode"`
			Silent      bool   `json:"disable_notification,omitempty"`
			LinkPreview struct {
				Disabled bool `json:"is_disabled"`
			} `json:"link_preview_options"`
		}{ChatID: p.text("chatId"), Topic: topic, Text: text, ParseMode: "HTML", Silent: p.on("silent")}
		m.LinkPreview.Disabled = true
		body, err := json.Marshal(m)
		if err != nil {
			return Outbound{}, fmt.Errorf("encode the Telegram message: %w", err)
		}
		return Outbound{Post: &Request{Method: http.MethodPost, URL: u, Header: jsonHeader(), Body: body}}, nil
	}
	parts := []part{{name: "chat_id", body: []byte(p.text("chatId"))}}
	if topic != 0 {
		parts = append(parts, part{name: "message_thread_id", body: []byte(strconv.Itoa(topic))})
	}
	parts = append(parts, part{name: "caption", body: []byte(text)}, part{name: "parse_mode", body: []byte("HTML")})
	if p.on("silent") {
		parts = append(parts, part{name: "disable_notification", body: []byte("true")})
	}
	body, ctype := multipartBody(append(parts, posterPart("photo", p.Poster)))
	return Outbound{Post: &Request{Method: http.MethodPost, URL: u, Header: http.Header{"Content-Type": {ctype}}, Body: body}}, nil
}

// telegramHTML is the title in bold, the lines and the link, in at most
// limit characters of text as Telegram counts it once the tags are parsed.
func telegramHTML(title string, lines []string, link string, limit int) string {
	reserve := 0
	if link != "" {
		reserve = 1 + len(linkLabel)
	}
	parts := fit(append([]string{cut(title, telegramTitle)}, lines...), 1, limit-reserve)
	var b strings.Builder
	for i, s := range parts {
		if i == 0 {
			b.WriteString("<b>" + escapeHTML(s) + "</b>")
			continue
		}
		b.WriteString("\n" + escapeHTML(s))
	}
	if link != "" {
		b.WriteString("\n<a href=\"" + escapeHTML(link) + "\">" + linkLabel + "</a>")
	}
	return b.String()
}

// telegramAnswer reads an answer whose ok is false by its error_code: a
// flood wait by parameters.retry_after, and an unknown chat.
func telegramAnswer(r Response, now time.Time) (Verdict, bool) {
	var a struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if json.Unmarshal(r.Body, &a) != nil || a.OK {
		return Verdict{}, false
	}
	switch {
	case a.ErrorCode == http.StatusTooManyRequests && a.Parameters.RetryAfter > 0:
		return slowDown(now, time.Duration(a.Parameters.RetryAfter)*time.Second, a.ErrorCode), true
	case a.ErrorCode == http.StatusBadRequest && strings.Contains(strings.ToLower(a.Description), "chat not found"):
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: ChatNotFound, Status: a.ErrorCode, Field: "chatId"}}, true
	case a.ErrorCode == http.StatusForbidden:
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: Unauthorized, Status: a.ErrorCode, Field: "chatId"}}, true
	}
	return Verdict{}, false
}
