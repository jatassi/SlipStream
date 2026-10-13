package search

import (
	"github.com/libp2p/go-yamux/v5"
	"golang.org/x/net/http2/h2c"

	"example.com/app/internal/link"
)

var _ = yamux.Server
var _ = h2c.NewHandler
var _ link.Key
