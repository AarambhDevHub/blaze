//go:build ignore

package main

import (
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Basic WebSocket endpoint
	app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
		defer ws.Close()

		// Send welcome message
		ws.WriteJSON(blaze.Map{
			"type":    "welcome",
			"message": "Connected to WebSocket",
			"time":    time.Now(),
		})

		// Message loop
		for {
			var msg blaze.Map
			if err := ws.ReadJSON(&msg); err != nil {
				log.Printf("WebSocket read error: %v", err)
				break
			}

			log.Printf("Received: %+v from %s", msg, ws.RemoteAddr())

			// Echo message back
			response := blaze.Map{
				"type":      "echo",
				"original":  msg,
				"timestamp": time.Now(),
				"client":    ws.RemoteAddr(),
			}

			if err := ws.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
				break
			}
		}
	})

	// Serve static files for testing
	app.GET("/", func(c *blaze.Context) error {
		return c.HTML(`
<!DOCTYPE html>
<html>
<head>
    <title>WebSocket Test</title>
</head>
<body>
    <div id="messages"></div>
    <input type="text" id="messageInput" placeholder="Enter message">
    <button onclick="sendMessage()">Send</button>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        const messages = document.getElementById('messages');
        
        ws.onopen = function(event) {
            addMessage('Connected to WebSocket');
        };
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            addMessage('Received: ' + JSON.stringify(data, null, 2));
        };
        
        ws.onclose = function(event) {
            addMessage('Connection closed');
        };
        
        ws.onerror = function(error) {
            addMessage('Error: ' + error);
        };
        
        function sendMessage() {
            const input = document.getElementById('messageInput');
            if (input.value) {
                ws.send(JSON.stringify({
                    message: input.value,
                    timestamp: new Date().toISOString()
                }));
                input.value = '';
            }
        }
        
        function addMessage(message) {
            const div = document.createElement('div');
            div.innerHTML = '<pre>' + message + '</pre>';
            messages.appendChild(div);
        }
        
        document.getElementById('messageInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') {
                sendMessage();
            }
        });
    </script>
</body>
</html>`)
	})

	log.Printf("🚀 WebSocket server starting on http://localhost:8080")
	log.Fatal(app.ListenAndServe())
}
