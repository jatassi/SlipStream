package mux

import (
	"github.com/hashicorp/yamux"
	goyamux "github.com/libp2p/go-yamux/v5"
	"golang.org/x/net/http2"
)

var _, _, _ = yamux.Server, goyamux.Server, http2.Server{}
