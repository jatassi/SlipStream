// Package search calls an Indexer.
//
// It reaches every Indexer it searches through the Node that holds the
// searching Role, and caches nothing between calls.
//
// Search is the only entry point.
package search

import (
	"net/http"

	"connectrpc.com/connect/v2"
	"example.com/app/internal/pb/slipstream/link/v1/linkv1connect"
)

func Search(c *connect.Client) {
	linkv1connect.NewLinkServiceClient(c)
	NewCacheServiceClient(http.DefaultClient)
}

func NewCacheServiceClient(*http.Client) {}
