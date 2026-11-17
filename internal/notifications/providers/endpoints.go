package providers

import "github.com/jatassi/slipstream/internal/node/endpoint"

// Endpoints are the fixed hosts of the providers that have one.
type Endpoints struct {
	Telegram   endpoint.URL `yaml:"telegram"`
	Pushover   endpoint.URL `yaml:"pushover"`
	Pushbullet endpoint.URL `yaml:"pushbullet"`
	// Ntfy is the server an ntfy connection with no server URL sends to.
	Ntfy endpoint.URL `yaml:"ntfy"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		Telegram:   endpoint.MustParseURL("https://api.telegram.org"),
		Pushover:   endpoint.MustParseURL("https://api.pushover.net"),
		Pushbullet: endpoint.MustParseURL("https://api.pushbullet.com"),
		Ntfy:       endpoint.MustParseURL("https://ntfy.sh"),
	}
}
