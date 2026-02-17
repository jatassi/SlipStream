package downloader

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/downloader/aria2"
	"github.com/slipstream/slipstream/internal/downloader/deluge"
	"github.com/slipstream/slipstream/internal/downloader/downloadstation"
	"github.com/slipstream/slipstream/internal/downloader/flood"
	"github.com/slipstream/slipstream/internal/downloader/freeboxdownload"
	"github.com/slipstream/slipstream/internal/downloader/hadouken"
	"github.com/slipstream/slipstream/internal/downloader/mock"
	"github.com/slipstream/slipstream/internal/downloader/qbittorrent"
	"github.com/slipstream/slipstream/internal/downloader/rqbit"
	"github.com/slipstream/slipstream/internal/downloader/rtorrent"
	"github.com/slipstream/slipstream/internal/downloader/sabnzbd"
	"github.com/slipstream/slipstream/internal/downloader/transmission"
	"github.com/slipstream/slipstream/internal/downloader/tribler"
	"github.com/slipstream/slipstream/internal/downloader/types"
	"github.com/slipstream/slipstream/internal/downloader/utorrent"
	"github.com/slipstream/slipstream/internal/downloader/vuze"
)

// NewClient creates a new download client of the specified type.
// Returns the client interface so callers can use polymorphism.
func NewClient(clientType ClientType, config *ClientConfig) (Client, error) { //nolint:gocyclo // switch over all client types
	switch clientType {
	case ClientTypeTransmission:
		return transmission.NewFromConfig(config), nil
	case ClientTypeQBittorrent:
		return qbittorrent.NewFromConfig(config), nil
	case ClientTypeDeluge:
		return deluge.NewFromConfig(config), nil
	case ClientTypeVuze:
		return vuze.NewFromConfig(config), nil
	case ClientTypeFlood:
		return flood.NewFromConfig(config), nil
	case ClientTypeAria2:
		return aria2.NewFromConfig(config), nil
	case ClientTypeSABnzbd:
		return sabnzbd.NewFromConfig(config), nil
	case ClientTypeRTorrent:
		return rtorrent.NewFromConfig(config), nil
	case ClientTypeUTorrent:
		return utorrent.NewFromConfig(config), nil
	case ClientTypeHadouken:
		return hadouken.NewFromConfig(config), nil
	case ClientTypeDownloadStation:
		return downloadstation.NewFromConfig(config), nil
	case ClientTypeFreeboxDownload:
		return freeboxdownload.NewFromConfig(config), nil
	case ClientTypeRQBit:
		return rqbit.NewFromConfig(config), nil
	case ClientTypeTribler:
		return tribler.NewFromConfig(config), nil
	case ClientTypeMock:
		return mock.NewFromConfig(config), nil
	case ClientTypeNZBGet:
		return nil, fmt.Errorf("%w: %s client not yet implemented", ErrUnsupportedClient, clientType)
	default:
		return nil, fmt.Errorf("%w: unknown client type %s", ErrUnsupportedClient, clientType)
	}
}

// NewTorrentClient creates a new torrent client of the specified type.
// Returns the TorrentClient interface for torrent-specific operations.
func NewTorrentClient(clientType ClientType, config *ClientConfig) (TorrentClient, error) { //nolint:gocyclo // switch over all client types
	if ProtocolForClient(clientType) != ProtocolTorrent {
		return nil, fmt.Errorf("%w: %s is not a torrent client", ErrUnsupportedClient, clientType)
	}

	switch clientType {
	case ClientTypeTransmission:
		return transmission.NewFromConfig(config), nil
	case ClientTypeQBittorrent:
		return qbittorrent.NewFromConfig(config), nil
	case ClientTypeDeluge:
		return deluge.NewFromConfig(config), nil
	case ClientTypeVuze:
		return vuze.NewFromConfig(config), nil
	case ClientTypeFlood:
		return flood.NewFromConfig(config), nil
	case ClientTypeAria2:
		return aria2.NewFromConfig(config), nil
	case ClientTypeRTorrent:
		return rtorrent.NewFromConfig(config), nil
	case ClientTypeUTorrent:
		return utorrent.NewFromConfig(config), nil
	case ClientTypeHadouken:
		return hadouken.NewFromConfig(config), nil
	case ClientTypeDownloadStation:
		return downloadstation.NewFromConfig(config), nil
	case ClientTypeFreeboxDownload:
		return freeboxdownload.NewFromConfig(config), nil
	case ClientTypeRQBit:
		return rqbit.NewFromConfig(config), nil
	case ClientTypeTribler:
		return tribler.NewFromConfig(config), nil
	case ClientTypeMock:
		return mock.NewFromConfig(config), nil
	default:
		return nil, fmt.Errorf("%w: unknown torrent client type %s", ErrUnsupportedClient, clientType)
	}
}

// NewUsenetClient creates a new usenet client of the specified type.
// Returns the UsenetClient interface for usenet-specific operations.
func NewUsenetClient(clientType ClientType, config *ClientConfig) (UsenetClient, error) {
	if ProtocolForClient(clientType) != ProtocolUsenet {
		return nil, fmt.Errorf("%w: %s is not a usenet client", ErrUnsupportedClient, clientType)
	}

	switch clientType {
	case ClientTypeSABnzbd:
		return sabnzbd.NewFromConfig(config), nil
	case ClientTypeNZBGet:
		return nil, fmt.Errorf("%w: %s client not yet implemented", ErrUnsupportedClient, clientType)
	default:
		return nil, fmt.Errorf("%w: unknown usenet client type %s", ErrUnsupportedClient, clientType)
	}
}

// ClientFromDownloadClient creates a Client from a DownloadClient database model.
func ClientFromDownloadClient(dc *DownloadClient) (Client, error) {
	return NewClient(ClientType(dc.Type), downloadClientToConfig(dc))
}

// TorrentClientFromDownloadClient creates a TorrentClient from a DownloadClient database model.
func TorrentClientFromDownloadClient(dc *DownloadClient) (TorrentClient, error) {
	return NewTorrentClient(ClientType(dc.Type), downloadClientToConfig(dc))
}

func downloadClientToConfig(dc *DownloadClient) *types.ClientConfig {
	return &types.ClientConfig{
		Host:     dc.Host,
		Port:     dc.Port,
		Username: dc.Username,
		Password: dc.Password,
		UseSSL:   dc.UseSSL,
		APIKey:   dc.APIKey,
		Category: dc.Category,
		URLBase:  dc.URLBase,
	}
}

// SupportedClientTypes returns a list of all supported client types.
func SupportedClientTypes() []ClientType {
	return []ClientType{
		ClientTypeTransmission,
		ClientTypeQBittorrent,
		ClientTypeDeluge,
		ClientTypeRTorrent,
		ClientTypeVuze,
		ClientTypeAria2,
		ClientTypeFlood,
		ClientTypeUTorrent,
		ClientTypeHadouken,
		ClientTypeDownloadStation,
		ClientTypeFreeboxDownload,
		ClientTypeRQBit,
		ClientTypeTribler,
		ClientTypeSABnzbd,
		ClientTypeNZBGet,
	}
}

// ImplementedClientTypes returns a list of client types that are fully implemented.
func ImplementedClientTypes() []ClientType {
	return []ClientType{
		ClientTypeTransmission,
		ClientTypeQBittorrent,
		ClientTypeDeluge,
		ClientTypeRTorrent,
		ClientTypeVuze,
		ClientTypeFlood,
		ClientTypeAria2,
		ClientTypeUTorrent,
		ClientTypeHadouken,
		ClientTypeDownloadStation,
		ClientTypeFreeboxDownload,
		ClientTypeRQBit,
		ClientTypeTribler,
		ClientTypeSABnzbd,
		ClientTypeMock,
	}
}

// IsClientTypeSupported returns true if the client type is recognized.
func IsClientTypeSupported(clientType string) bool {
	for _, ct := range SupportedClientTypes() {
		if string(ct) == clientType {
			return true
		}
	}
	return false
}

// IsClientTypeImplemented returns true if the client type is fully implemented.
func IsClientTypeImplemented(clientType string) bool {
	for _, ct := range ImplementedClientTypes() {
		if string(ct) == clientType {
			return true
		}
	}
	return false
}
