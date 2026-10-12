package nodesapi

import "example.com/app/internal/api"

func Routes() {
	api.Register()
	func() {
		api.Register()
	}()
}

func listRoutes() {
	api.Register()
}

type server struct{}

func (server) Routes() {
	api.Register()
}

var register = api.Register

var _ = func() int { api.RegisterStream(); return 0 }()
