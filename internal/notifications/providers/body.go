package providers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
)

// bodyVersion is the Webhook body's version. A field is only ever added,
// and a change a receiver can't ignore is a new version.
const bodyVersion = 1

// webhookBody is the Webhook's body, version 1, which is also a Custom
// Script's event file.
type webhookBody struct {
	Version int              `json:"version"`
	Event   entry.Kind       `json:"event"`
	At      time.Time        `json:"at"`
	Title   string           `json:"title"`
	Lines   []string         `json:"lines"`
	Link    string           `json:"link,omitempty"`
	Image   string           `json:"image,omitempty"`
	Work    *entry.BodyWork  `json:"work,omitempty"`
	Files   []entry.BodyFile `json:"files,omitempty"`
	Health  []entry.BodyItem `json:"health,omitempty"`
}

func bodyOf(p Payload) []byte {
	b := webhookBody{
		Version: bodyVersion,
		Event:   p.Kind,
		At:      p.At.UTC(),
		Title:   p.Message.Title,
		Lines:   p.Message.Lines,
		Link:    p.Message.Link.Path,
		Work:    p.Message.Body.Work,
		Files:   p.Message.Body.Files,
		Health:  p.Message.Body.Health,
	}
	if b.Lines == nil {
		b.Lines = []string{}
	}
	if p.Poster != nil {
		b.Image = p.Poster.URL
	}
	out, err := json.Marshal(b)
	if err != nil {
		panic(fmt.Sprintf("providers: the webhook body doesn't encode: %v", err))
	}
	return out
}
