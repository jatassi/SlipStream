package sim

import (
	"testing"

	"example.com/app/internal/link"
	"example.com/app/internal/pb/slipstream/link/v1/linkv1connect"
	"example.com/app/internal/pb/slipstream/rpctest/v1/rpctestv1connect"
)

func TestMismatch(t *testing.T) {
	linkv1connect.NewLinkServiceClient(nil)
	rpctestv1connect.NewPingServiceClient(nil)
	_ = link.Key{}
}
