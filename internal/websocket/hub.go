package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// incomingMessage wraps a message from a client.
type incomingMessage struct {
	client  *Client
	message []byte
}

// DevModePayload is the payload for devmode:set messages.
type DevModePayload struct {
	Enabled bool `json:"enabled"`
}

// Hub manages WebSocket connections and broadcasts.
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	incoming     chan incomingMessage
	mu           sync.RWMutex
	onDevModeSet func(enabled bool) error
	logger       *zerolog.Logger
}

// Client represents a WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Message represents a WebSocket message.
type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *zerolog.Logger) *Hub {
	subLogger := logger.With().Str("component", "websocket").Logger()
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		incoming:   make(chan incomingMessage, 256),
		logger:     &subLogger,
	}
}

// SetDevModeHandler registers a handler for dev mode toggle messages.
func (h *Hub) SetDevModeHandler(handler func(enabled bool) error) {
	h.onDevModeSet = handler
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case incoming := <-h.incoming:
			h.handleIncoming(incoming)
		}
	}
}

// handleIncoming processes messages received from clients.
func (h *Hub) handleIncoming(incoming incomingMessage) {
	var msg Message
	if err := json.Unmarshal(incoming.message, &msg); err != nil {
		return
	}

	if msg.Type == "devmode:set" {
		if h.onDevModeSet == nil {
			return
		}

		payloadBytes, err := json.Marshal(msg.Payload)
		if err != nil {
			return
		}

		var payload DevModePayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return
		}

		if err := h.onDevModeSet(payload.Enabled); err != nil {
			h.Broadcast("devmode:error", map[string]interface{}{
				"error":   err.Error(),
				"enabled": !payload.Enabled,
			})
			return
		}

		h.Broadcast("devmode:changed", map[string]interface{}{
			"enabled": payload.Enabled,
		})
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	msg := Message{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error().
			Err(err).
			Str("msgType", msgType).
			Msg("Failed to marshal WebSocket message")
		return
	}
	h.broadcast <- data
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket handles WebSocket connection upgrade.
func (h *Hub) HandleWebSocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	return nil
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			return err
		}
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) { //nolint:revive,staticcheck // Error intentionally ignored, checked but not logged
				// Unexpected WebSocket close error (expected errors are ignored)
			}
			break
		}

		// Forward message to hub for processing
		c.hub.incoming <- incomingMessage{
			client:  c,
			message: message,
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !c.handleSendMessage(message, ok) {
				return
			}

		case <-ticker.C:
			if !c.sendPing() {
				return
			}
		}
	}
}

func (c *Client) handleSendMessage(message []byte, ok bool) bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return false
	}
	if !ok {
		_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
		return false
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return false
	}

	// Drain queued messages
	n := len(c.send)
	for i := 0; i < n; i++ {
		if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
			return false
		}
	}
	return true
}

func (c *Client) sendPing() bool {
	if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return false
	}
	return c.conn.WriteMessage(websocket.PingMessage, nil) == nil
}

// UpdateStatus represents the current state of the auto-update system.
type UpdateStatus struct {
	State          string  `json:"state"`
	CurrentVersion string  `json:"currentVersion"`
	LatestVersion  string  `json:"latestVersion,omitempty"`
	Progress       float64 `json:"progress"`
	DownloadedMB   float64 `json:"downloadedMB,omitempty"`
	TotalMB        float64 `json:"totalMB,omitempty"`
	Error          string  `json:"error,omitempty"`
}

// UpdateStatusProvider is implemented by services that can provide update status.
type UpdateStatusProvider interface {
	GetUpdateStatus() *UpdateStatus
}

// BroadcastUpdateStatus sends an update status message to all connected clients.
func (h *Hub) BroadcastUpdateStatus(status interface{}) {
	h.Broadcast("update:status", status)
}
