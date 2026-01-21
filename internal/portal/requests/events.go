package requests

import (
	"time"
)

const (
	EventRequestCreated = "request:created"
	EventRequestUpdated = "request:updated"
	EventRequestDeleted = "request:deleted"
)

type RequestCreatedPayload struct {
	RequestID int64  `json:"requestId"`
	UserID    int64  `json:"userId"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type RequestUpdatedPayload struct {
	RequestID    int64      `json:"requestId"`
	UserID       int64      `json:"userId"`
	MediaType    string     `json:"mediaType"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	PreviousStatus string   `json:"previousStatus,omitempty"`
	MediaID      *int64     `json:"mediaId,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type RequestDeletedPayload struct {
	RequestID int64  `json:"requestId"`
	UserID    int64  `json:"userId"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
}

type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

type EventBroadcaster struct {
	hub Broadcaster
}

func NewEventBroadcaster(hub Broadcaster) *EventBroadcaster {
	return &EventBroadcaster{hub: hub}
}

func (b *EventBroadcaster) BroadcastRequestCreated(request *Request) {
	if b.hub == nil {
		return
	}
	b.hub.Broadcast(EventRequestCreated, RequestCreatedPayload{
		RequestID: request.ID,
		UserID:    request.UserID,
		MediaType: request.MediaType,
		Title:     request.Title,
		Status:    request.Status,
		CreatedAt: request.CreatedAt,
	})
}

func (b *EventBroadcaster) BroadcastRequestUpdated(request *Request, previousStatus string) {
	if b.hub == nil {
		return
	}
	b.hub.Broadcast(EventRequestUpdated, RequestUpdatedPayload{
		RequestID:      request.ID,
		UserID:         request.UserID,
		MediaType:      request.MediaType,
		Title:          request.Title,
		Status:         request.Status,
		PreviousStatus: previousStatus,
		MediaID:        request.MediaID,
		UpdatedAt:      request.UpdatedAt,
	})
}

func (b *EventBroadcaster) BroadcastRequestDeleted(request *Request) {
	if b.hub == nil {
		return
	}
	b.hub.Broadcast(EventRequestDeleted, RequestDeletedPayload{
		RequestID: request.ID,
		UserID:    request.UserID,
		MediaType: request.MediaType,
		Title:     request.Title,
	})
}
