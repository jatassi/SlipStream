package downloader

import (
	"fmt"

	"github.com/slipstream/slipstream/internal/downloader/mock"
	"github.com/slipstream/slipstream/internal/downloader/qbittorrent"
	"github.com/slipstream/slipstream/internal/downloader/sabnzbd"
	"github.com/slipstream/slipstream/internal/downloader/transmission"
	"github.com/slipstream/slipstream/internal/downloader/types"
)

// NewClient creates a new download client of the specified type.
// Returns the client interface so callers can use polymorphism.
func NewClient(clientType ClientType, config *ClientConfig) (Client, error) {
	switch clientType {
	case ClientTypeTransmission:
		return transmission.NewFromConfig(config), nil
	case ClientTypeQBittorrent:
		return qbittorrent.NewFromConfig(config), nil
	case ClientTypeSABnzbd:
		return sabnzbd.NewFromConfig(config), nil
	case ClientTypeMock:
		return mock.NewFromConfig(config), nil
	case ClientTypeDeluge, ClientTypeRTorrent, ClientTypeNZBGet:
		return nil, fmt.Errorf("%w: %s client not yet implemented", ErrUnsupportedClient, clientType)
	default:
		return nil, fmt.Errorf("%w: unknown client type %s", ErrUnsupportedClient, clientType)
	}
}

// NewTorrentClient creates a new torrent client of the specified type.
// Returns the TorrentClient interface for torrent-specific operations.
func NewTorrentClient(clientType ClientType, config *ClientConfig) (TorrentClient, error) {
	if ProtocolForClient(clientType) != ProtocolTorrent {
		return nil, fmt.Errorf("%w: %s is not a torrent client", ErrUnsupportedClient, clientType)
	}

	switch clientType {
	case ClientTypeTransmission:
		return transmission.NewFromConfig(config), nil
	case ClientTypeQBittorrent:
		return qbittorrent.NewFromConfig(config), nil
	case ClientTypeMock:
		return mock.NewFromConfig(config), nil
	case ClientTypeDeluge, ClientTypeRTorrent:
		return nil, fmt.Errorf("%w: %s client not yet implemented", ErrUnsupportedClient, clientType)
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
	config := &types.ClientConfig{
		Host:     dc.Host,
		Port:     dc.Port,
		Username: dc.Username,
		Password: dc.Password,
		UseSSL:   dc.UseSSL,
		Category: dc.Category,
	}

	return NewClient(ClientType(dc.Type), config)
}

// TorrentClientFromDownloadClient creates a TorrentClient from a DownloadClient database model.
func TorrentClientFromDownloadClient(dc *DownloadClient) (TorrentClient, error) {
	config := &types.ClientConfig{
		Host:     dc.Host,
		Port:     dc.Port,
		Username: dc.Username,
		Password: dc.Password,
		UseSSL:   dc.UseSSL,
		Category: dc.Category,
	}

	return NewTorrentClient(ClientType(dc.Type), config)
}

// SupportedClientTypes returns a list of all supported client types.
func SupportedClientTypes() []ClientType {
	return []ClientType{
		ClientTypeTransmission,
		ClientTypeQBittorrent,
		ClientTypeDeluge,
		ClientTypeRTorrent,
		ClientTypeSABnzbd,
		ClientTypeNZBGet,
	}
}

// ImplementedClientTypes returns a list of client types that are fully implemented.
func ImplementedClientTypes() []ClientType {
	return []ClientType{
		ClientTypeTransmission,
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
