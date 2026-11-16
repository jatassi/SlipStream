package providers

import "github.com/jatassi/slipstream/internal/node/endpoint"

// Endpoints are the fixed hosts of the providers that have one.
type Endpoints struct {
	Telegram   endpoint.URL `yaml:"telegram"`
	Pushover   endpoint.URL `yaml:"pushover"`
	Pushbullet endpoint.URL `yaml:"pushbullet"`
	Ntfy       endpoint.URL `yaml:"ntfy"`
}
