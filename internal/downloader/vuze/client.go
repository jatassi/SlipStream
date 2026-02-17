package vuze

import (
	"context"
	"fmt"

	"github.com/slipstream/slipstream/internal/downloader/transmission"
	"github.com/slipstream/slipstream/internal/downloader/types"
)

// Client wraps the Transmission client since Vuze uses the Transmission RPC protocol.
type Client struct {
	*transmission.Client
}

var _ types.TorrentClient = (*Client)(nil)

func NewFromConfig(cfg *types.ClientConfig) *Client {
	return &Client{
		Client: transmission.NewFromConfig(cfg),
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeVuze
}

func (c *Client) Test(ctx context.Context) error {
	session, err := c.GetSessionInfo()
	if err != nil {
		return err
	}

	if rpcVersion, ok := session["rpc-version"].(float64); ok {
		if int(rpcVersion) < 14 {
			return fmt.Errorf("vuze rpc-version %d is below minimum required version 14", int(rpcVersion))
		}
	}

	return nil
}
