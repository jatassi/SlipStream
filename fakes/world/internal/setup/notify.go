package setup

import (
	"context"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// NotifyWorld is what the world gives the notify step: the Controller's
// notify-sink front, its capture-script front, and the capture-script
// program the run built.
type NotifyWorld struct {
	Sink, Report, Script string
}

// Notify adds the Scenario's notification connections on the Controller
// as an Admin would, each on, with posters and its recommended events.
// A typed address is the Controller's notify-sink front, and the fixed
// hosts reach the sink through Endpoints.Notify.
func Notify(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, w NotifyWorld) error {
	controller := slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Setup == scenario.SetupController })
	for _, id := range s.Notify {
		p, _ := providers.ByID(id)
		nc := nodeapi.NewConnection{Name: p.Label, Provider: string(id), On: true, Posters: true}
		switch id {
		case "discord":
			nc.Secrets = map[string]string{"webhookUrl": w.Sink + "/api/webhooks/1180000000000000000/discord-token"}
		case "telegram":
			nc.Values, nc.Secrets = map[string]any{"chatId": "-1001234567890"}, map[string]string{"botToken": "123456:telegram-token"}
		case "slack":
			nc.Secrets = map[string]string{"webhookUrl": w.Sink + "/services/T0000/B0000/slack-token"}
		case "pushover":
			nc.Secrets = map[string]string{"appToken": "pushover-app-token", "userKey": "pushover-user-key"}
		case "gotify":
			nc.Values, nc.Secrets = map[string]any{"serverUrl": w.Sink + "/gotify"}, map[string]string{"appToken": "gotify-app-token"}
		case "ntfy":
			nc.Secrets = map[string]string{"topic": "slipstream-ntfy-topic"}
		case "pushbullet":
			nc.Secrets = map[string]string{"accessToken": "o.pushbullet-token"}
		case "apprise":
			nc.Values, nc.Secrets = map[string]any{"serverUrl": w.Sink + "/apprise"}, map[string]string{"key": "apprise-key"}
		case "email":
			nc.Values = map[string]any{"addresses": []string{s.Admin.Email}}
		case "webhook":
			nc.Secrets = map[string]string{"url": w.Sink + "/hook/slipstream/webhook-token"}
		case "script":
			nc.Node = claims.Members[controller].ID
			nc.Values = map[string]any{"path": w.Script, "args": []string{"--report", w.Report}}
		}
		if err := nodeapi.AddNotificationConnection(ctx, c, url(ready[controller]), claims.APIKey, nc); err != nil {
			return err
		}
	}
	return nil
}
