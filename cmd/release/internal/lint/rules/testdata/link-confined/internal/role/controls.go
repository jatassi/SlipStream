package role

import (
	"connectrpc.com/connect/v2"

	"example.com/app/internal/link"
	"example.com/app/internal/pb/slipstream/node/v1/nodev1connect"
)

func serve(srv *connect.Server, l *link.Link) { nodev1connect.RegisterSyncServiceHandler(srv, nil) }
