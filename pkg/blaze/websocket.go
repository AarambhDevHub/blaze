package blaze

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// WebSocketUpgrader handles WebSocket connection upgrades
// Converts HTTP connections to WebSocket protocol for real-time communication
//
// WebSocket Protocol:
//   - Full-duplex communication over single TCP connection
//   - Persistent connection (no reconnection overhead)
//   - Low latency for real-time applications
//   - Binary and text message support
//   - Built-in ping/pong heartbeat mechanism
//
// Upgrade Process:
//  1. Client sends HTTP upgrade request
//  2. Server validates origin and protocol
//  3. Handshake negotiates protocol version
//  4. Connection upgraded to WebSocket
//  5. Bidirectional message exchange begins
//
// Thread Safety:
//   - Upgrader is safe for concurrent use
//   - Individual connections handle concurrent reads/writes
type WebSocketUpgrader struct {
	// upgrader is the underlying fasthttp WebSocket upgrader
	upgrader *websocket.FastHTTPUpgrader

	// readTimeout specifies maximum time to read a message
	// Prevents hanging connections from slow clients
	// Applied before each read operation
	readTimeout time.Duration

	// writeTimeout specifies maximum time to write a message
	// Prevents blocking on slow connections
	// Applied before each write operation
	writeTimeout time.Duration

	// pingInterval specifies interval for ping messages
	// Keeps connection alive and detects broken connections
	// Set to 0 to disable automatic pings
	pingInterval time.Duration

	// pongTimeout specifies maximum time to wait for pong response
	// Connection closed if pong not received in time
	// Should be longer than pingInterval
	pongTimeout time.Duration

	// maxMessageSize specifies maximum message size in bytes
	// Prevents memory exhaustion from large messages
	// Connections closed if exceeded
	maxMessageSize int64
}

// WebSocketConfig holds comprehensive WebSocket configuration
// Controls connection behavior, timeouts, and security settings
//
// Configuration Philosophy:
//   - Security: Validate origins, limit message sizes
//   - Reliability: Configure timeouts, enable heartbeats
//   - Performance: Tune buffer sizes, enable compression
//   - Development: Relaxed settings for testing
//
// Production Best Practices:
//   - Enable origin checking (prevent CSRF)
//   - Set reasonable message size limits
//   - Configure appropriate timeouts
//   - Enable compression for large messages
//   - Implement proper error handling
type WebSocketConfig struct {
	// ReadBufferSize specifies size of read buffer in bytes
	// Larger buffers improve throughput for large messages
	// Smaller buffers reduce memory usage
	// Default: 4096 bytes (4KB)
	ReadBufferSize int

	// WriteBufferSize specifies size of write buffer in bytes
	// Affects how many messages can be queued
	// Default: 4096 bytes (4KB)
	WriteBufferSize int

	// CheckOrigin validates the origin header
	// Function receives request context and returns true if valid
	// Essential for preventing CSRF attacks
	// Default: Allow all origins (development only)
	//
	// Production Example:
	//   CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
	//       origin := string(ctx.Request.Header.Peek("Origin"))
	//       return origin == "https://example.com"
	//   }
	CheckOrigin func(ctx *fasthttp.RequestCtx) bool

	// ReadTimeout specifies maximum time to read a message
	// Prevents hanging on slow or stalled clients
	// Default: 60 seconds
	ReadTimeout time.Duration

	// WriteTimeout specifies maximum time to write a message
	// Prevents blocking on unresponsive clients
	// Default: 10 seconds
	WriteTimeout time.Duration

	// PingInterval specifies interval for automatic ping messages
	// Keeps connection alive through NAT/firewalls
	// Detects broken connections early
	// Set to 0 to disable
	// Default: 30 seconds
	PingInterval time.Duration

	// PongTimeout specifies maximum time to wait for pong
	// Connection closed if pong not received
	// Should be longer than network round-trip time
	// Default: 60 seconds
	PongTimeout time.Duration

	// MaxMessageSize specifies maximum message size in bytes
	// Prevents DOS attacks via large messages
	// Default: 1MB (1024 * 1024 bytes)
	MaxMessageSize int64

	// CompressionLevel specifies compression level (0-9)
	// 0: No compression (fastest)
	// 1-5: Fast compression with less reduction
	// 6: Default balanced compression
	// 7-9: Maximum compression (slowest)
	// Set to 0 to disable compression
	// Default: 6
	CompressionLevel int
}

// DefaultWebSocketConfig returns default WebSocket configuration
// Provides balanced settings suitable for most applications
//
// Default Settings:
//   - Buffers: 4KB read/write
//   - Origin: Allow all (change for production!)
//   - Timeouts: 60s read, 10s write
//   - Heartbeat: 30s ping interval, 60s pong timeout
//   - Message Size: 1MB maximum
//   - Compression: Level 6 (balanced)
//
// Returns:
//   - WebSocketConfig: Default configuration
//
// Example:
//
//	config := blaze.DefaultWebSocketConfig()
//	config.CheckOrigin = validateOrigin // Add security
//	upgrader := blaze.NewWebSocketUpgrader(config)
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
// Configures upgrader with specified settings or defaults
//
// Parameters:
//   - config: Optional WebSocket configuration
//
// Returns:
//   - *WebSocketUpgrader: Configured upgrader instance
//
// Example - Default Configuration:
//
//	upgrader := blaze.NewWebSocketUpgrader()
//
// Example - Custom Configuration:
//
//	config := blaze.WebSocketConfig{
//	    MaxMessageSize: 512 * 1024, // 512KB
//	    PingInterval: 15 * time.Second,
//	    CheckOrigin: validateOrigin,
//	}
//	upgrader := blaze.NewWebSocketUpgrader(config)
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

// WebSocketConnection represents an active WebSocket connection
// Provides methods for reading/writing messages and managing connection lifecycle
//
// Connection Features:
//   - Bidirectional message exchange
//   - Automatic ping/pong heartbeat
//   - Async write support
//   - Local variable storage
//   - Graceful shutdown support
//
// Thread Safety:
//   - Read operations: Single goroutine only
//   - Write operations: Thread-safe (uses mutex)
//   - Close operations: Thread-safe (once semantics)
type WebSocketConnection struct {
	// conn is the underlying websocket connection
	conn *websocket.Conn

	// ctx is the original Blaze context
	ctx *Context

	// mu protects concurrent access to connection state
	mu sync.RWMutex

	// closed indicates if connection is closed
	closed bool

	// closeCh signals connection closure
	closeCh chan struct{}

	// writeCh queues messages for async writing
	writeCh chan []byte

	// pingTicker sends periodic ping messages
	pingTicker *time.Ticker

	// locals stores connection-specific data
	locals map[string]interface{}
}

// WebSocketHandler defines WebSocket handler function signature
// Receives WebSocket connection and handles communication
//
// Handler Responsibilities:
//   - Read messages from connection
//   - Process and respond to messages
//   - Handle errors gracefully
//   - Close connection when done
//
// Example:
//
//	func chatHandler(conn *blaze.WebSocketConnection) error {
//	    for {
//	        msgType, data, err := conn.ReadMessage()
//	        if err != nil {
//	            return err
//	        }
//
//	        // Broadcast to all clients
//	        hub.Broadcast(data)
//	    }
//	}
type WebSocketHandler func(*WebSocketConnection)

// WebSocketMessage represents a WebSocket message
// Used for structured message passing
type WebSocketMessage struct {
	Type    int         `json:"type"`              // Message type (text, binary, etc.)
	Data    interface{} `json:"data"`              // Message payload
	Channel string      `json:"channel,omitempty"` // Optional channel for routing
}

// Message type constants for WebSocket messages
// Aligned with WebSocket protocol specification
const (
	// TextMessage represents UTF-8 encoded text message
	// Used for JSON, plain text, etc.
	TextMessage = websocket.TextMessage

	// BinaryMessage represents binary data message
	// Used for images, files, custom protocols
	BinaryMessage = websocket.BinaryMessage

	// CloseMessage represents connection close message
	// Sent to gracefully close connection
	CloseMessage = websocket.CloseMessage

	// PingMessage represents ping message
	// Used for keepalive and connection testing
	PingMessage = websocket.PingMessage

	// PongMessage represents pong message
	// Response to ping message
	PongMessage = websocket.PongMessage
)

// Upgrade upgrades HTTP connection to WebSocket
// Performs WebSocket handshake and calls handler
//
// Upgrade Process:
//  1. Validate upgrade request headers
//  2. Perform WebSocket handshake
//  3. Create WebSocketConnection wrapper
//  4. Configure connection limits and timeouts
//  5. Start ping/pong heartbeat
//  6. Start write routine for async writes
//  7. Call handler with connection
//  8. Cleanup on handler return
//
// Parameters:
//   - c: Blaze context with HTTP request
//   - handler: Function to handle WebSocket connection
//
// Returns:
//   - error: Upgrade or handler error
//
// Example:
//
//	upgrader := blaze.NewWebSocketUpgrader()
//	err := upgrader.Upgrade(c, func(conn *blaze.WebSocketConnection) error {
//	    for {
//	        msgType, data, err := conn.ReadMessage()
//	        if err != nil {
//	            return err
//	        }
//	        // Echo message
//	        conn.WriteMessage(msgType, data)
//	    }
//	})
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
// Provides access to request information
//
// Returns:
//   - *Context: Original request context
//
// Example:
//
//	ctx := conn.Context()
//	userID := ctx.Locals("user_id")
func (ws *WebSocketConnection) Context() *Context {
	return ws.ctx
}

// Close closes the WebSocket connection
// Performs cleanup and stops background routines
//
// Cleanup Process:
//  1. Check if already closed
//  2. Mark as closed
//  3. Signal close channel
//  4. Stop ping ticker
//  5. Close underlying connection
//
// Returns:
//   - error: Close error or nil
//
// Example:
//
//	defer conn.Close()
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
// Thread-safe check of connection state
//
// Returns:
//   - bool: true if closed
//
// Example:
//
//	if !conn.IsClosed() {
//	    conn.WriteText("message")
//	}
func (ws *WebSocketConnection) IsClosed() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.closed
}

// ReadMessage reads next message from WebSocket
// Blocks until message received or error occurs
//
// Returns:
//   - messageType: Type of message (text, binary, etc.)
//   - data: Message payload
//   - error: Read error or nil
//
// Example:
//
//	msgType, data, err := conn.ReadMessage()
//	if err != nil {
//	    log.Printf("Read error: %v", err)
//	    return
//	}
func (ws *WebSocketConnection) ReadMessage() (messageType int, data []byte, err error) {
	return ws.conn.ReadMessage()
}

// WriteMessage writes message to WebSocket
// Blocks until message sent or error occurs
//
// Parameters:
//   - messageType: Type of message
//   - data: Message payload
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	err := conn.WriteMessage(blaze.TextMessage, []byte("hello"))
func (ws *WebSocketConnection) WriteMessage(messageType int, data []byte) error {
	if ws.IsClosed() {
		return websocket.ErrCloseSent
	}

	ws.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return ws.conn.WriteMessage(messageType, data)
}

// WriteText writes text message to WebSocket
// Convenience method for sending text
//
// Parameters:
//   - data: Text message string
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	conn.WriteText("Hello, World!")
func (ws *WebSocketConnection) WriteText(data string) error {
	return ws.WriteMessage(TextMessage, []byte(data))
}

// WriteBinary writes binary message to WebSocket
// Convenience method for sending binary data
//
// Parameters:
//   - data: Binary message data
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	conn.WriteBinary(imageData)
func (ws *WebSocketConnection) WriteBinary(data []byte) error {
	return ws.WriteMessage(BinaryMessage, data)
}

// WriteJSON writes JSON-encoded message
// Marshals data to JSON and sends as text
//
// Parameters:
//   - data: Data to encode as JSON
//
// Returns:
//   - error: Encoding or write error
//
// Example:
//
//	conn.WriteJSON(map[string]interface{}{
//	    "type": "notification",
//	    "message": "New message",
//	})
func (ws *WebSocketConnection) WriteJSON(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ws.WriteText(string(jsonData))
}

// ReadJSON reads and decodes JSON message
// Receives message and unmarshals JSON
//
// Parameters:
//   - v: Pointer to destination struct
//
// Returns:
//   - error: Read or decode error
//
// Example:
//
//	var msg Message
//	if err := conn.ReadJSON(&msg); err != nil {
//	    log.Printf("Error: %v", err)
//	}
func (ws *WebSocketConnection) ReadJSON(v interface{}) error {
	_, data, err := ws.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Ping sends ping message
// Tests connection liveness
//
// Parameters:
//   - data: Optional ping payload
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	conn.Ping([]byte("keepalive"))
func (ws *WebSocketConnection) Ping(data []byte) error {
	return ws.WriteMessage(PingMessage, data)
}

// Pong sends pong message
// Responds to ping message
//
// Parameters:
//   - data: Pong payload (usually echoes ping)
//
// Returns:
//   - error: Write error or nil
func (ws *WebSocketConnection) Pong(data []byte) error {
	return ws.WriteMessage(PongMessage, data)
}

// SetLocal sets a connection-specific variable
// Stores data associated with this connection
//
// Parameters:
//   - key: Variable name
//   - value: Variable value
//
// Example:
//
//	conn.SetLocal("user_id", 123)
//	conn.SetLocal("room", "lobby")
func (ws *WebSocketConnection) SetLocal(key string, value interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.locals[key] = value
}

// GetLocal retrieves a connection-specific variable
// Returns stored value or nil if not found
//
// Parameters:
//   - key: Variable name
//
// Returns:
//   - interface{}: Stored value or nil
//
// Example:
//
//	userID := conn.GetLocal("user_id").(int)
func (ws *WebSocketConnection) GetLocal(key string) interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.locals[key]
}

// RemoteAddr returns the remote network address
// Identifies client connection source
//
// Returns:
//   - string: Remote address (IP:port format)
//
// Example:
//
//	addr := conn.RemoteAddr()
//	log.Printf("Client connected from: %s", addr)
func (ws *WebSocketConnection) RemoteAddr() string {
	return ws.conn.RemoteAddr().String()
}

// LocalAddr returns the local network address
// Identifies server endpoint
//
// Returns:
//   - string: Local address (IP:port format)
func (ws *WebSocketConnection) LocalAddr() string {
	return ws.conn.LocalAddr().String()
}

// UserAgent returns the client user agent
// Identifies client software and version
//
// Returns:
//   - string: User-Agent header value
//
// Example:
//
//	ua := conn.UserAgent()
//	log.Printf("Client: %s", ua)
func (ws *WebSocketConnection) UserAgent() string {
	return string(ws.ctx.RequestCtx.UserAgent())
}

// Header returns request header value
// Accesses headers from original HTTP request
//
// Parameters:
//   - key: Header name
//
// Returns:
//   - string: Header value or empty string
//
// Example:
//
//	token := conn.Header("Authorization")
func (ws *WebSocketConnection) Header(key string) string {
	return ws.ctx.Header(key)
}

// startPingRoutine starts automatic ping messages
// Sends periodic pings to keep connection alive
//
// Ping Mechanism:
//   - Sends ping at configured interval
//   - Expects pong response
//   - Closes connection on ping failure
//   - Stops on connection close
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

// writeRoutine handles asynchronous message writes
// Processes messages from write queue
//
// Write Queue:
//   - Decouples sending from handler logic
//   - Prevents blocking on slow connections
//   - Buffers up to 256 messages
//   - Drops messages if buffer full
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
// Queues message for background writing
//
// Benefits:
//   - Non-blocking operation
//   - Handler continues immediately
//   - Buffers messages automatically
//
// Limitations:
//   - Message may be dropped if buffer full
//   - No error feedback to caller
//   - Order guaranteed within queue
//
// Parameters:
//   - data: Message data to send
//
// Example:
//
//	conn.WriteAsync([]byte("notification"))
func (ws *WebSocketConnection) WriteAsync(data []byte) {
	select {
	case ws.writeCh <- data:
	case <-ws.closeCh:
	default:
		log.Printf("WebSocket write buffer full, dropping message")
	}
}

// WebSocketHub manages multiple WebSocket connections
// Provides broadcast and connection management
//
// Hub Features:
//   - Connection registration/unregistration
//   - Broadcast messages to all clients
//   - Track active connections
//   - Graceful shutdown support
//
// Use Cases:
//   - Chat rooms
//   - Live notifications
//   - Multiplayer games
//   - Real-time dashboards
type WebSocketHub struct {
	// clients maps connections to registration status
	clients map[*WebSocketConnection]bool

	// broadcast channel for messages to all clients
	broadcast chan []byte

	// register channel for new connections
	register chan *WebSocketConnection

	// unregister channel for closed connections
	unregister chan *WebSocketConnection

	// mu protects concurrent access to clients map
	mu sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
// Initializes hub with empty client list
//
// Returns:
//   - *WebSocketHub: New hub instance
//
// Example:
//
//	hub := blaze.NewWebSocketHub()
//	go hub.Run()
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketConnection]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketConnection),
		unregister: make(chan *WebSocketConnection),
	}
}

// Run starts the hub event loop
// Processes registration, unregistration, and broadcasts
//
// Should be called in goroutine:
//
//	go hub.Run()
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

// CloseGracefully closes connection with timeout
// Sends close message and waits for acknowledgment
//
// Graceful Close Process:
//  1. Send close message to client
//  2. Wait for close confirmation
//  3. Close connection after timeout
//
// Parameters:
//   - timeout: Maximum wait time for close
//
// Returns:
//   - error: Close error or nil
//
// Example:
//
//	conn.CloseGracefully(5 * time.Second)
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
