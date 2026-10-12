package media

import (
	"example.com/app/internal/api"
	vendorapi "example.com/vendor/api"
)

type registry struct{}

func (registry) Register() {}

func Load() {
	vendorapi.Register()
	api := registry{}
	api.Register()
}
