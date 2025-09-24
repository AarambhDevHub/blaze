# WebSockets

Blaze provides comprehensive WebSocket support for real-time, bidirectional communication between clients and servers. Built on top of the high-performance `fasthttp/websocket` library, it offers features like connection management, message handling, broadcasting, and automatic ping/pong mechanisms.

## Overview

WebSocket connections in Blaze allow upgrading HTTP connections to WebSocket protocol, enabling full-duplex communication. The framework provides an easy-to-use API with advanced features like connection pooling, automatic heartbeat, and graceful shutdown handling.

### Key Features

- **HTTP to WebSocket Upgrade**: Seamless connection upgrade from HTTP requests
- **Message Types**: Support for text, binary, JSON, ping, and pong messages
- **Connection Management**: Built-in connection pooling and lifecycle management
- **Broadcasting**: Hub-based message broadcasting to multiple clients
- **Automatic Ping/Pong**: Configurable heartbeat mechanism for connection health
- **Graceful Shutdown**: Integration with the framework's shutdown system
- **Middleware Support**: Route-level middleware for WebSocket endpoints

## Basic Usage

### Simple WebSocket Handler

```go
package main

import (
    "log"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()
    
    // Basic WebSocket endpoint
    app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
        for {
            // Read message from client
            messageType, data, err := ws.ReadMessage()
            if err != nil {
                log.Printf("Read error: %v", err)
                break
            }
            
            // Echo message back to client
            if err := ws.WriteMessage(messageType, data); err != nil {
                log.Printf("Write error: %v", err)
                break
            }
        }
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### JSON Message Handling

```go
type ChatMessage struct {
    Username string `json:"username"`
    Message  string `json:"message"`
    Time     string `json:"time"`
}

app.WebSocket("/chat", func(ws *blaze.WebSocketConnection) {
    for {
        var msg ChatMessage
        
        // Read JSON message
        if err := ws.ReadJSON(&msg); err != nil {
            log.Printf("JSON read error: %v", err)
            break
        }
        
        // Process message
        msg.Time = time.Now().Format(time.RFC3339)
        
        // Send JSON response
        if err := ws.WriteJSON(msg); err != nil {
            log.Printf("JSON write error: %v", err)
            break
        }
    }
})
```

## Configuration

### WebSocket Configuration

```go
config := &blaze.WebSocketConfig{
    ReadBufferSize:   8192,                    // Read buffer size
    WriteBufferSize:  8192,                    // Write buffer size
    ReadTimeout:      60 * time.Second,        // Read timeout
    WriteTimeout:     10 * time.Second,        // Write timeout
    PingInterval:     30 * time.Second,        // Ping interval
    PongTimeout:      60 * time.Second,        // Pong timeout
    MaxMessageSize:   1024 * 1024,             // 1MB max message size
    CompressionLevel: 6,                       // Compression level
    CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
        // Custom origin validation
        origin := string(ctx.Request.Header.Peek("Origin"))
        return strings.HasSuffix(origin, ".example.com")
    },
}

app.WebSocketWithConfig("/ws", handler, config)
```

### Default Configuration

```go
// Default WebSocket configuration
config := blaze.DefaultWebSocketConfig()
// Returns:
// - ReadBufferSize: 4096
// - WriteBufferSize: 4096
// - ReadTimeout: 60 seconds
// - WriteTimeout: 10 seconds
// - PingInterval: 30 seconds
// - PongTimeout: 60 seconds
// - MaxMessageSize: 1MB
// - CompressionLevel: 6
// - CheckOrigin: allows all origins
```

## Connection Management

### Connection Properties

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    // Access connection properties
    log.Printf("Remote address: %s", ws.RemoteAddr())
    log.Printf("Local address: %s", ws.LocalAddr())
    log.Printf("User agent: %s", ws.UserAgent())
    
    // Access request headers
    token := ws.Header("Authorization")
    
    // Access underlying context
    ctx := ws.Context()
    userID := ctx.Query("user_id")
    
    // Set local variables
    ws.SetLocal("user_id", userID)
    ws.SetLocal("connected_at", time.Now())
    
    // Get local variables
    if uid := ws.GetLocal("user_id"); uid != nil {
        log.Printf("User ID: %s", uid.(string))
    }
})
```

### Connection Lifecycle

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("WebSocket panic: %v", r)
        }
        ws.Close()
    }()
    
    // Connection established
    log.Printf("Client connected: %s", ws.RemoteAddr())
    
    for {
        if ws.IsClosed() {
            break
        }
        
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            log.Printf("Read error: %v", err)
            break
        }
        
        // Handle message
        handleMessage(ws, messageType, data)
    }
    
    // Connection closed
    log.Printf("Client disconnected: %s", ws.RemoteAddr())
})
```

## Message Types

### Text Messages

```go
// Send text message
err := ws.WriteText("Hello, WebSocket!")

// Read text message
messageType, data, err := ws.ReadMessage()
if err == nil && messageType == blaze.TextMessage {
    text := string(data)
    log.Printf("Received: %s", text)
}
```

### Binary Messages

```go
// Send binary data
binaryData := []byte{0x00, 0x01, 0x02, 0x03}
err := ws.WriteBinary(binaryData)

// Read binary message
messageType, data, err := ws.ReadMessage()
if err == nil && messageType == blaze.BinaryMessage {
    log.Printf("Received binary data: %x", data)
}
```

### JSON Messages

```go
type Response struct {
    Status string      `json:"status"`
    Data   interface{} `json:"data"`
}

// Send JSON
response := Response{
    Status: "success",
    Data:   map[string]string{"message": "Hello"},
}
err := ws.WriteJSON(response)

// Read JSON
var received Response
err := ws.ReadJSON(&received)
```

### Ping/Pong Messages

```go
// Manual ping
err := ws.Ping([]byte("ping"))

// Manual pong
err := ws.Pong([]byte("pong"))

// Automatic ping is handled by the framework
// Configure with PingInterval in WebSocketConfig
```

## Broadcasting with WebSocket Hub

### Creating a Hub

```go
package main

import (
    "log"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

var hub = blaze.NewWebSocketHub()

func main() {
    app := blaze.New()
    
    // Start the hub in a goroutine
    go hub.Run()
    
    app.WebSocket("/ws", handleWebSocket)
    
    // REST endpoint to broadcast messages
    app.POST("/broadcast", func(c *blaze.Context) error {
        var msg struct {
            Message string `json:"message"`
        }
        
        if err := c.BindJSON(&msg); err != nil {
            return c.Status(400).JSON(blaze.Map{"error": "Invalid JSON"})
        }
        
        // Broadcast to all connected clients
        hub.Broadcast([]byte(msg.Message))
        
        return c.JSON(blaze.Map{"success": true})
    })
    
    log.Fatal(app.ListenAndServe())
}

func handleWebSocket(ws *blaze.WebSocketConnection) {
    // Register client with hub
    hub.Register(ws)
    defer hub.Unregister(ws)
    
    // Handle incoming messages
    for {
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            break
        }
        
        // Broadcast message to all clients
        hub.Broadcast(data)
    }
}
```

### Hub Statistics

```go
// Get number of connected clients
clientCount := hub.GetClientCount()
log.Printf("Connected clients: %d", clientCount)

// Get all connected clients
clients := hub.GetClients()
for _, client := range clients {
    log.Printf("Client: %s", client.RemoteAddr())
}
```

## Advanced Features

### Custom Message Routing

```go
type WebSocketMessage struct {
    Type    string      `json:"type"`
    Channel string      `json:"channel"`
    Data    interface{} `json:"data"`
}

app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    for {
        var msg WebSocketMessage
        if err := ws.ReadJSON(&msg); err != nil {
            break
        }
        
        switch msg.Type {
        case "join":
            handleJoinChannel(ws, msg.Channel)
        case "leave":
            handleLeaveChannel(ws, msg.Channel)
        case "message":
            handleChannelMessage(ws, msg.Channel, msg.Data)
        default:
            ws.WriteJSON(WebSocketMessage{
                Type: "error",
                Data: "Unknown message type",
            })
        }
    }
})
```

### Authentication and Authorization

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    // Check authentication
    token := ws.Header("Authorization")
    if token == "" {
        ws.WriteJSON(map[string]string{
            "error": "Authentication required",
        })
        ws.Close()
        return
    }
    
    // Validate token
    user, err := validateToken(token)
    if err != nil {
        ws.WriteJSON(map[string]string{
            "error": "Invalid token",
        })
        ws.Close()
        return
    }
    
    // Store user info
    ws.SetLocal("user", user)
    
    // Continue with message handling
    handleMessages(ws)
})
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    // Create rate limiter (10 messages per second)
    limiter := rate.NewLimiter(rate.Limit(10), 20)
    
    for {
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            break
        }
        
        // Check rate limit
        if !limiter.Allow() {
            ws.WriteJSON(map[string]string{
                "error": "Rate limit exceeded",
            })
            continue
        }
        
        // Process message
        handleMessage(ws, messageType, data)
    }
})
```

## Route Groups and Middleware

### WebSocket Routes in Groups

```go
// Create API group
api := app.Group("/api/v1")

// Add WebSocket to group
api.WebSocket("/ws", handleWebSocket)
api.WebSocketWithConfig("/chat", handleChat, chatConfig)
```

### Middleware Integration

```go
// Custom WebSocket middleware
func WSAuthMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Check authentication before upgrading
            token := c.Header("Authorization")
            if token == "" {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Authentication required",
                })
            }
            
            // Validate token
            if !isValidToken(token) {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Invalid token",
                })
            }
            
            // Store user info in context
            user := getUserFromToken(token)
            c.SetLocals("user", user)
            
            return next(c)
        }
    }
}

// Apply middleware to WebSocket route
wsGroup := app.Group("/ws")
wsGroup.Use(WSAuthMiddleware())
wsGroup.WebSocket("/chat", handleChat)
```

## Error Handling

### Connection Errors

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("WebSocket panic: %v", r)
            ws.Close()
        }
    }()
    
    for {
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, 
                websocket.CloseGoingAway, 
                websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }
        
        // Handle message with error recovery
        if err := processMessage(ws, messageType, data); err != nil {
            log.Printf("Message processing error: %v", err)
            
            // Send error response
            ws.WriteJSON(map[string]string{
                "error": err.Error(),
            })
        }
    }
})
```

### Graceful Shutdown

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    ctx := ws.Context()
    
    for {
        // Check if server is shutting down
        if ctx.IsShuttingDown() {
            ws.WriteText("Server is shutting down")
            ws.Close()
            break
        }
        
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            break
        }
        
        // Process message
        handleMessage(ws, messageType, data)
    }
})
```

## Performance Optimization

### Asynchronous Writing

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    // Use async writing for better performance
    for {
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            break
        }
        
        // Non-blocking write
        ws.WriteAsync(data)
    }
})
```

### Connection Pooling

```go
type ConnectionPool struct {
    connections map[string]*blaze.WebSocketConnection
    mutex       sync.RWMutex
}

func (p *ConnectionPool) Add(id string, conn *blaze.WebSocketConnection) {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    p.connections[id] = conn
}

func (p *ConnectionPool) Remove(id string) {
    p.mutex.Lock()
    defer p.mutex.Unlock()
    delete(p.connections, id)
}

func (p *ConnectionPool) Broadcast(message []byte) {
    p.mutex.RLock()
    defer p.mutex.RUnlock()
    
    for _, conn := range p.connections {
        conn.WriteAsync(message)
    }
}
```

## Examples

### Real-time Chat Application

```go
package main

import (
    "encoding/json"
    "log"
    "time"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

type ChatRoom struct {
    clients    map[*blaze.WebSocketConnection]bool
    broadcast  chan []byte
    register   chan *blaze.WebSocketConnection
    unregister chan *blaze.WebSocketConnection
}

type Message struct {
    Username string    `json:"username"`
    Content  string    `json:"content"`
    Time     time.Time `json:"time"`
    Type     string    `json:"type"`
}

func NewChatRoom() *ChatRoom {
    return &ChatRoom{
        clients:    make(map[*blaze.WebSocketConnection]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *blaze.WebSocketConnection),
        unregister: make(chan *blaze.WebSocketConnection),
    }
}

func (room *ChatRoom) Run() {
    for {
        select {
        case client := <-room.register:
            room.clients[client] = true
            
            // Send welcome message
            welcome := Message{
                Type:    "system",
                Content: "Welcome to the chat!",
                Time:    time.Now(),
            }
            data, _ := json.Marshal(welcome)
            client.WriteAsync(data)
            
        case client := <-room.unregister:
            if _, ok := room.clients[client]; ok {
                delete(room.clients, client)
                client.Close()
            }
            
        case message := <-room.broadcast:
            for client := range room.clients {
                select {
                case client.writeCh <- message:
                default:
                    delete(room.clients, client)
                    client.Close()
                }
            }
        }
    }
}

func main() {
    app := blaze.New()
    room := NewChatRoom()
    
    go room.Run()
    
    app.WebSocket("/chat", func(ws *blaze.WebSocketConnection) {
        room.register <- ws
        defer func() {
            room.unregister <- ws
        }()
        
        for {
            var msg Message
            if err := ws.ReadJSON(&msg); err != nil {
                break
            }
            
            msg.Time = time.Now()
            data, _ := json.Marshal(msg)
            room.broadcast <- data
        }
    })
    
    log.Fatal(app.ListenAndServe())
}
```

This comprehensive WebSocket documentation covers all aspects of using WebSockets in the Blaze framework, from basic usage to advanced features like broadcasting and connection management.