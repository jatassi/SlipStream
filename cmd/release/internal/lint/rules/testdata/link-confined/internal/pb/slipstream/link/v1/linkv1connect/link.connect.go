package linkv1connect

import "connectrpc.com/connect/v2"

func NewLinkServiceClient(c *connect.Client) LinkServiceClient { return nil }

var Probe = NewLinkServiceClient(nil)
