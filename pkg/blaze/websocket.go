package blaze

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// WebSocketUpgrader handles WebSocket connections
type WebSocketUpgrader struct {
	upgrader       *websocket.FastHTTPUpgrader
	readTimeout    time.Duration
	writeTimeout   time.Duration
	pingInterval   time.Duration
	pongTimeout    time.Duration
	maxMessageSize int64
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	ReadBufferSize   int
	WriteBufferSize  int
	CheckOrigin      func(ctx *fasthttp.RequestCtx) bool
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PingInterval     time.Duration
	PongTimeout      time.Duration
	MaxMessageSize   int64
	CompressionLevel int
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		CheckOrigin:      func(ctx *fasthttp.RequestCtx) bool { return true },
		ReadTimeout:      60 * time.Second,
		WriteTimeout:     10 * time.Second,
		PingInterval:     30 * time.Second,
		PongTimeout:      60 * time.Second,
		MaxMessageSize:   1024 * 1024, // 1MB
		CompressionLevel: 6,
	}
}

// NewWebSocketUpgrader creates a new WebSocket upgrader
func NewWebSocketUpgrader(config ...*WebSocketConfig) *WebSocketUpgrader {
	var cfg *WebSocketConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	} else {
		cfg = DefaultWebSocketConfig()
	}

	upgrader := &websocket.FastHTTPUpgrader{
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		CheckOrigin:       cfg.CheckOrigin,
		EnableCompression: cfg.CompressionLevel > 0,
	}

	return &WebSocketUpgrader{
		upgrader:       upgrader,
		readTimeout:    cfg.ReadTimeout,
		writeTimeout:   cfg.WriteTimeout,
		pingInterval:   cfg.PingInterval,
		pongTimeout:    cfg.PongTimeout,
		maxMessageSize: cfg.MaxMessageSize,
	}
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	conn       *websocket.Conn
	ctx        *Context
	mu         sync.RWMutex
	closed     bool
	closeCh    chan struct{}
	writeCh    chan []byte
	pingTicker *time.Ticker
	locals     map[string]interface{}
}

// WebSocketHandler defines WebSocket handler function signature
type WebSocketHandler func(*WebSocketConnection)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    int         `json:"type"`
	Data    interface{} `json:"data"`
	Channel string      `json:"channel,omitempty"`
}

// MessageType constants for WebSocket messages
const (
	TextMessage   = websocket.TextMessage
	BinaryMessage = websocket.BinaryMessage
	CloseMessage  = websocket.CloseMessage
	PingMessage   = websocket.PingMessage
	PongMessage   = websocket.PongMessage
)

// Upgrade upgrades HTTP connection to WebSocket
func (wu *WebSocketUpgrader) Upgrade(c *Context, handler WebSocketHandler) error {
	err := wu.upgrader.Upgrade(c.RequestCtx, func(conn *websocket.Conn) {
		wsConn := &WebSocketConnection{
			conn:    conn,
			ctx:     c,
			closed:  false,
			closeCh: make(chan struct{}),
			writeCh: make(chan []byte, 256),
			locals:  make(map[string]interface{}),
		}

		// Set connection limits
		conn.SetReadLimit(wu.maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(wu.readTimeout))
		conn.SetWriteDeadline(time.Now().Add(wu.writeTimeout))

		// Set pong handler
		conn.SetPongHandler(func(data string) error {
			conn.SetReadDeadline(time.Now().Add(wu.pongTimeout))
			return nil
		})

		// Start ping routine
		if wu.pingInterval > 0 {
			wsConn.startPingRoutine()
		}

		// Start write routine
		go wsConn.writeRoutine()

		// Handle the connection
		handler(wsConn)

		wsConn.Close()
	})

	return err
}

// Context returns the underlying Blaze context
func (ws *WebSocketConnection) Context() *Context {
	return ws.ctx
}

// Close closes the WebSocket connection
func (ws *WebSocketConnection) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return nil
	}

	ws.closed = true
	close(ws.closeCh)

	if ws.pingTicker != nil {
		ws.pingTicker.Stop()
	}

	return ws.conn.Close()
}

// IsClosed returns true if connection is closed
func (ws *WebSocketConnection) IsClosed() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.closed
}

// ReadMessage reads a message from WebSocket
func (ws *WebSocketConnection) ReadMessage() (messageType int, data []byte, err error) {
	return ws.conn.ReadMessage()
}

// WriteMessage writes a message to WebSocket
func (ws *WebSocketConnection) WriteMessage(messageType int, data []byte) error {
	if ws.IsClosed() {
		return websocket.ErrCloseSent
	}

	ws.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ws.conn.WriteMessage(messageType, data)
}

// WriteText writes a text message to WebSocket
func (ws *WebSocketConnection) WriteText(data string) error {
	return ws.WriteMessage(TextMessage, []byte(data))
}

// WriteBinary writes a binary message to WebSocket
func (ws *WebSocketConnection) WriteBinary(data []byte) error {
	return ws.WriteMessage(BinaryMessage, data)
}

// WriteJSON writes a JSON message to WebSocket
func (ws *WebSocketConnection) WriteJSON(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteText(string(jsonData))
}

// ReadJSON reads a JSON message from WebSocket
func (ws *WebSocketConnection) ReadJSON(v interface{}) error {
	_, data, err := ws.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Ping sends a ping message
func (ws *WebSocketConnection) Ping(data []byte) error {
	return ws.WriteMessage(PingMessage, data)
}

// Pong sends a pong message
func (ws *WebSocketConnection) Pong(data []byte) error {
	return ws.WriteMessage(PongMessage, data)
}

// SetLocal sets a local variable for the WebSocket connection
func (ws *WebSocketConnection) SetLocal(key string, value interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.locals[key] = value
}

// GetLocal gets a local variable from the WebSocket connection
func (ws *WebSocketConnection) GetLocal(key string) interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.locals[key]
}

// RemoteAddr returns the remote address
func (ws *WebSocketConnection) RemoteAddr() string {
	return ws.conn.RemoteAddr().String()
}

// LocalAddr returns the local address
func (ws *WebSocketConnection) LocalAddr() string {
	return ws.conn.LocalAddr().String()
}

// UserAgent returns the user agent
func (ws *WebSocketConnection) UserAgent() string {
	return string(ws.ctx.RequestCtx.UserAgent())
}

// Header returns request header value
func (ws *WebSocketConnection) Header(key string) string {
	return ws.ctx.Header(key)
}

// startPingRoutine starts the ping routine
func (ws *WebSocketConnection) startPingRoutine() {
	ws.pingTicker = time.NewTicker(30 * time.Second)
	go func() {
		defer ws.pingTicker.Stop()
		for {
			select {
			case <-ws.pingTicker.C:
				if err := ws.Ping([]byte("ping")); err != nil {
					log.Printf("WebSocket ping failed: %v", err)
					ws.Close()
					return
				}
			case <-ws.closeCh:
				return
			}
		}
	}()
}

// writeRoutine handles asynchronous writes
func (ws *WebSocketConnection) writeRoutine() {
	for {
		select {
		case data := <-ws.writeCh:
			if err := ws.WriteText(string(data)); err != nil {
				log.Printf("WebSocket write failed: %v", err)
				ws.Close()
				return
			}
		case <-ws.closeCh:
			return
		}
	}
}

// WriteAsync writes message asynchronously
func (ws *WebSocketConnection) WriteAsync(data []byte) {
	select {
	case ws.writeCh <- data:
	case <-ws.closeCh:
	default:
		log.Printf("WebSocket write buffer full, dropping message")
	}
}

// WebSocketHub manages multiple WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketConnection]bool
	broadcast  chan []byte
	register   chan *WebSocketConnection
	unregister chan *WebSocketConnection
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketConnection]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketConnection),
		unregister: make(chan *WebSocketConnection),
	}
}

// Run starts the hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected: %s", client.RemoteAddr())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected: %s", client.RemoteAddr())

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.writeCh <- message:
				default:
					delete(h.clients, client)
					client.Close()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register registers a client
func (h *WebSocketHub) Register(client *WebSocketConnection) {
	h.register <- client
}

// Unregister unregisters a client
func (h *WebSocketHub) Unregister(client *WebSocketConnection) {
	h.unregister <- client
}

// Broadcast broadcasts a message to all clients
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClients returns all connected clients
func (h *WebSocketHub) GetClients() []*WebSocketConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*WebSocketConnection, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// Close with graceful shutdown support
func (ws *WebSocketConnection) CloseGracefully(timeout time.Duration) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return nil
	}

	// Send close message
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server shutting down")
	ws.conn.SetWriteDeadline(time.Now().Add(timeout))
	ws.conn.WriteMessage(websocket.CloseMessage, closeMessage)

	// Wait for close confirmation or timeout
	ws.conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		_, _, err := ws.conn.ReadMessage()
		if err != nil {
			break
		}
	}

	ws.closed = true
	close(ws.closeCh)

	if ws.pingTicker != nil {
		ws.pingTicker.Stop()
	}

	return ws.conn.Close()
}

// Update WebSocketHub with graceful shutdown
func (h *WebSocketHub) Shutdown(timeout time.Duration) error {
	log.Println("🔌 Shutting down WebSocket hub...")

	h.mu.RLock()
	clients := make([]*WebSocketConnection, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	// Close all connections gracefully
	for _, client := range clients {
		go func(c *WebSocketConnection) {
			if err := c.CloseGracefully(timeout); err != nil {
				log.Printf("Error closing WebSocket connection: %v", err)
			}
		}(client)
	}

	// Wait for connections to close or timeout
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.GetClientCount() == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ WebSocket hub shutdown complete. Remaining connections: %d", h.GetClientCount())
	return nil
}
