// Package downloader provides download client abstractions and implementations.
package downloader

import (
	"github.com/slipstream/slipstream/internal/downloader/types"
)

// Re-export types for convenience.
// This allows external packages to use downloader.Client instead of types.Client.

type (
	Protocol          = types.Protocol
	ClientType        = types.ClientType
	ClientConfig      = types.ClientConfig
	Client            = types.Client
	TorrentClient     = types.TorrentClient
	UsenetClient      = types.UsenetClient
	AddOptions        = types.AddOptions
	DownloadItem      = types.DownloadItem
	Status            = types.Status
	TorrentInfo       = types.TorrentInfo
	UsenetQueueItem   = types.UsenetQueueItem
	UsenetHistoryItem = types.UsenetHistoryItem
)

// Re-export constants.
const (
	ProtocolTorrent = types.ProtocolTorrent
	ProtocolUsenet  = types.ProtocolUsenet

	ClientTypeTransmission = types.ClientTypeTransmission
	ClientTypeQBittorrent  = types.ClientTypeQBittorrent
	ClientTypeDeluge       = types.ClientTypeDeluge
	ClientTypeRTorrent     = types.ClientTypeRTorrent
	ClientTypeSABnzbd      = types.ClientTypeSABnzbd
	ClientTypeNZBGet       = types.ClientTypeNZBGet

	StatusQueued      = types.StatusQueued
	StatusDownloading = types.StatusDownloading
	StatusPaused      = types.StatusPaused
	StatusCompleted   = types.StatusCompleted
	StatusSeeding     = types.StatusSeeding
	StatusError       = types.StatusError
	StatusUnknown     = types.StatusUnknown
)

// Re-export errors.
var (
	ErrNotImplemented = types.ErrNotImplemented
	ErrNotConnected   = types.ErrNotConnected
	ErrAuthFailed     = types.ErrAuthFailed
	ErrNotFound       = types.ErrNotFound
)

// Re-export functions.
var ProtocolForClient = types.ProtocolForClient
