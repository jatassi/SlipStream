package liveapi

import httpapi "example.com/app/internal/api"

func Routes() {
	httpapi.RegisterStream()
	httpapi.Register[int, string, bool]()
	register := httpapi.Register
	register()
}
