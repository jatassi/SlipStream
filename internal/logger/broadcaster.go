package logger

import (
	"encoding/json"
	"sync"
)

const defaultBufferSize = 1000

// Broadcaster is the interface for broadcasting messages.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{})
}

// LogEntry represents a parsed log entry for streaming.
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Component string         `json:"component,omitempty"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// LogBroadcaster implements io.Writer and broadcasts log entries to WebSocket.
type LogBroadcaster struct {
	hub        Broadcaster
	buffer     *RingBuffer[LogEntry]
	bufferSize int
	mu         sync.RWMutex
}

// NewLogBroadcaster creates a new log broadcaster.
// Hub can be nil initially and set later with SetHub.
func NewLogBroadcaster(hub Broadcaster, bufferSize int) *LogBroadcaster {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	return &LogBroadcaster{
		hub:        hub,
		buffer:     NewRingBuffer[LogEntry](bufferSize),
		bufferSize: bufferSize,
	}
}

// SetHub sets the broadcaster hub for sending messages.
func (b *LogBroadcaster) SetHub(hub Broadcaster) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hub = hub
}

// Write implements io.Writer. It receives JSON log entries from zerolog.
func (b *LogBroadcaster) Write(p []byte) (n int, err error) {
	n = len(p)

	entry, parseErr := b.parseLogEntry(p)
	if parseErr != nil {
		return n, nil //nolint:nilerr // Silently ignore malformed log entries
	}

	b.buffer.Push(entry)

	b.mu.RLock()
	hub := b.hub
	b.mu.RUnlock()

	if hub != nil {
		hub.Broadcast("logs:entry", entry)
	}

	return n, nil
}

// GetRecentLogs returns all buffered log entries.
func (b *LogBroadcaster) GetRecentLogs() []LogEntry {
	return b.buffer.GetAll()
}

// parseLogEntry parses a zerolog JSON entry into a LogEntry.
func (b *LogBroadcaster) parseLogEntry(data []byte) (LogEntry, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return LogEntry{}, err
	}

	entry := LogEntry{
		Fields: make(map[string]any),
	}

	if ts, ok := raw["time"].(string); ok {
		entry.Timestamp = ts
		delete(raw, "time")
	}

	if level, ok := raw["level"].(string); ok {
		entry.Level = level
		delete(raw, "level")
	}

	if component, ok := raw["component"].(string); ok {
		entry.Component = component
		delete(raw, "component")
	}

	if msg, ok := raw["message"].(string); ok {
		entry.Message = msg
		delete(raw, "message")
	}

	for k, v := range raw {
		entry.Fields[k] = v
	}

	return entry, nil
}
