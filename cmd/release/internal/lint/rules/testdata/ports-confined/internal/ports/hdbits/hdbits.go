package hdbits

import (
	"net"
	"net/http"

	_ "example.com/app/internal/cardigann"
	_ "example.com/app/internal/ports/internal/gazelle"
	_ "example.com/app/internal/ports/norbits"
	_ "example.com/app/internal/ports/port"
	_ "example.com/app/internal/searching"
)

var (
	method = http.MethodGet
	header = http.Header{}
	client = &http.Client{}
	dialer net.Dialer
)

func fetch() { _, _ = http.Get("https://hdbits.test/") }
