package search

import (
	"testing"

	"connectrpc.com/connect/v2/connecthttp"

	"example.com/app/internal/link"
	"example.com/app/internal/pb/slipstream/link/v1/linkv1connect"
)

func TestSearch(t *testing.T) {
	linkv1connect.NewLinkServiceClient(nil)
	_ = connecthttp.Mount
	_ = link.Key{}
}
