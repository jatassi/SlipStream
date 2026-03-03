package contracts

import "context"

// Broadcaster defines the WebSocket broadcast interface used across services.
type Broadcaster interface {
	Broadcast(msgType string, payload any)
}

// HealthService defines the health status registration interface.
// Uses the full 5-method variant (including SetWarningStr). Packages that
// only use a subset of these methods can accept this wider interface safely.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	ClearStatusStr(category, id string)
	SetWarningStr(category, id, message string)
}

// StatusChangeLogger defines the media status change logging interface.
type StatusChangeLogger interface {
	LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error
}

// FileDeleteHandler defines the file deletion callback interface.
type FileDeleteHandler interface {
	OnFileDeleted(ctx context.Context, mediaType string, fileID int64) error
}

// QueueTrigger defines the download queue trigger interface.
type QueueTrigger interface {
	Trigger()
}
