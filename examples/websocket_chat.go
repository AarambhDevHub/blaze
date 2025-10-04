//go:build ignore

package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Create WebSocket hub
	hub := blaze.NewWebSocketHub()
	go hub.Run()

	// Chat WebSocket endpoint
	app.WebSocket("/ws/chat", func(ws *blaze.WebSocketConnection) {
		// Register client
		hub.Register(ws)

		defer func() {
			hub.Unregister(ws)
		}()

		// Send welcome message
		ws.WriteJSON(blaze.Map{
			"type":    "system",
			"message": "Welcome to the chat!",
			"users":   hub.GetClientCount(),
		})

		// Notify others about new user
		systemMessage := blaze.Map{
			"type":    "system",
			"message": "A user joined the chat",
		}
		if data, err := json.Marshal(systemMessage); err == nil {
			hub.Broadcast(data)
		}

		// Message loop
		for {
			var msg blaze.Map
			if err := ws.ReadJSON(&msg); err != nil {
				log.Printf("WebSocket read error: %v", err)
				break
			}

			// Create chat message
			chatMessage := blaze.Map{
				"type":      "message",
				"message":   msg["message"],
				"user":      ws.RemoteAddr(),
				"timestamp": time.Now().Format(time.RFC3339),
			}

			// Convert to JSON and broadcast
			if data, err := json.Marshal(chatMessage); err == nil {
				hub.Broadcast(data)
			} else {
				log.Printf("JSON marshal error: %v", err)
			}
		}
	})

	// Status endpoint
	app.GET("/ws/chat/status", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"connected_clients": hub.GetClientCount(),
			"timestamp":         time.Now(),
		})
	})

	// Chat page
	app.GET("/", func(c *blaze.Context) error {
		return c.HTML(`
<!DOCTYPE html>
<html>
<head>
    <title>Blaze WebSocket Chat</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        #messages { border: 1px solid #ccc; height: 400px; overflow-y: scroll; padding: 10px; margin-bottom: 10px; }
        #messageInput { width: 70%; padding: 5px; }
        #sendButton { width: 25%; padding: 5px; }
        .message { margin: 5px 0; }
        .system { color: #666; font-style: italic; }
        .user-message { color: #333; }
        .timestamp { color: #999; font-size: 0.8em; }
    </style>
</head>
<body>
    <h1>Blaze WebSocket Chat</h1>
    <div id="messages"></div>
    <input type="text" id="messageInput" placeholder="Type your message...">
    <button id="sendButton">Send</button>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws/chat');
        const messages = document.getElementById('messages');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        
        ws.onopen = function(event) {
            addMessage('Connected to chat', 'system');
        };
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            
            if (data.type === 'system') {
                addMessage(data.message, 'system');
            } else if (data.type === 'message') {
                const timestamp = new Date(data.timestamp).toLocaleTimeString();
                addMessage(data.user + ': ' + data.message, 'user-message', timestamp);
            }
        };
        
        ws.onclose = function(event) {
            addMessage('Connection closed', 'system');
        };
        
        ws.onerror = function(error) {
            addMessage('Connection error', 'system');
        };
        
        function sendMessage() {
            const message = messageInput.value.trim();
            if (message && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    message: message,
                    timestamp: new Date().toISOString()
                }));
                messageInput.value = '';
            }
        }
        
        function addMessage(message, className = '', timestamp = '') {
            const div = document.createElement('div');
            div.className = 'message ' + className;
            
            let content = message;
            if (timestamp) {
                content += ' <span class="timestamp">(' + timestamp + ')</span>';
            }
            
            div.innerHTML = content;
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        }
        
        sendButton.addEventListener('click', sendMessage);
        
        messageInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    </script>
</body>
</html>`)
	})

	log.Printf("🚀 Chat server starting on http://localhost:8080")
	log.Fatal(app.ListenAndServe())
}
