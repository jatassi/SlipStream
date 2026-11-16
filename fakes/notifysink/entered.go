package notifysink

import (
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// Entered reads the server and webhook URLs an Admin typed into each
// notification connection: a URL setting as typed, or a secret URL by the
// host the API shows of it. The Controller dials them. The fixed hosts
// aren't typed, since Endpoints.Notify points them at the sink.
func Entered(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	body, err := get("/api/v1/notifications/connections")
	if err != nil {
		return nil, err
	}
	var list struct {
		Connections []struct {
			Name     string                     `json:"name"`
			Provider providers.ID               `json:"provider"`
			Values   map[string]json.RawMessage `json:"values"`
			Shown    map[string]string          `json:"shown"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listNotificationConnections: %w", err)
	}
	if len(list.Connections) == 0 {
		return nil, nil
	}
	controller, err := controllerOf(get)
	if err != nil {
		return nil, err
	}
	var out []fake.Entered
	for _, c := range list.Connections {
		p, ok := providers.ByID(c.Provider)
		if !ok {
			return nil, fmt.Errorf("connection %q has the unknown provider %s", c.Name, c.Provider)
		}
		for _, f := range p.Fields {
			if f.Kind != providers.URL {
				continue
			}
			var typed string
			_ = json.Unmarshal(c.Values[f.Key], &typed)
			if host := c.Shown[f.Key]; f.Shown == providers.ShownHost && host != "" {
				typed = "https://" + host
			}
			if typed != "" {
				out = append(out, fake.Entered{What: fmt.Sprintf("%s '%s' %s", p.Label, c.Name, f.Label), URL: typed, NodeID: controller})
			}
		}
	}
	return out, nil
}

func controllerOf(get func(path string) ([]byte, error)) (string, error) {
	body, err := get("/api/v1/nodes")
	if err != nil {
		return "", err
	}
	var list struct {
		Nodes []struct {
			ID   string `json:"id"`
			Self bool   `json:"self"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return "", fmt.Errorf("listNodes: %w", err)
	}
	for _, n := range list.Nodes {
		if n.Self {
			return n.ID, nil
		}
	}
	return "", fmt.Errorf("listNodes names no Controller")
}
