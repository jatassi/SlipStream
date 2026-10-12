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

func typedRoutes() {
	api.Register[int, string, bool]()
	api.RegisterStream[int]()
	wrap(api.Register)
}

func wrap(any) {}

func Routes2() {
	api.Register[int, string, bool]()
}
