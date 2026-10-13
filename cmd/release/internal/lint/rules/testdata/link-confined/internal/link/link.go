package link

import (
	"connectrpc.com/connect/v2"

	"example.com/app/internal/link/internal/mux"
	"example.com/app/internal/pb/slipstream/link/v1/linkv1connect"
)

func dial(c *connect.Client) { linkv1connect.NewLinkServiceClient(c); mux.Open() }
