package notifysink_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/notifysink"
)

func TestEnteredReadsTheURLsAnAdminTyped(t *testing.T) {
	answers := map[string]string{
		"/api/v1/notifications/connections": `{"connections":[
			{"name":"Family","provider":"discord","values":{"username":"SlipStream"},"shown":{"webhookUrl":"127.0.0.1:4100"}},
			{"name":"Home","provider":"gotify","values":{"serverUrl":"https://127.0.0.1:4100/gotify","priority":5},"shown":{}},
			{"name":"Phone","provider":"pushover","values":{},"shown":{}}
		]}`,
		"/api/v1/nodes": `{"nodes":[{"id":"n2","self":false},{"id":"n1","self":true}]}`,
	}
	got, err := notifysink.Entered(func(path string) ([]byte, error) {
		body, ok := answers[path]
		if !ok {
			return nil, fmt.Errorf("GET %s = 404", path)
		}
		return []byte(body), nil
	})
	want := []fake.Entered{
		{What: "Discord 'Family' Webhook URL", URL: "https://127.0.0.1:4100", NodeID: "n1"},
		{What: "Gotify 'Home' Server URL", URL: "https://127.0.0.1:4100/gotify", NodeID: "n1"},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Entered = %+v, %v, want %+v", got, err, want)
	}
}
