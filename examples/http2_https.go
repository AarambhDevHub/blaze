//go:build ignore

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	// Create app with production configuration
	config := blaze.ProductionConfig()
	config.Host = "127.0.0.1"
	config.Port = 3000
	config.TLSPort = 3443
	config.Development = true // Enable development mode for auto-generated certs

	app := blaze.NewWithConfig(config)

	// Configure TLS with auto-generation
	tlsConfig := blaze.DevelopmentTLSConfig()
	tlsConfig.Domains = []string{"localhost", "127.0.0.1"}
	app.SetTLSConfig(tlsConfig)

	// Configure HTTP/2
	http2Config := blaze.DefaultHTTP2Config()
	http2Config.MaxConcurrentStreams = 500
	http2Config.EnablePush = true
	app.SetHTTP2Config(http2Config)

	// Add HTTP/2 specific middleware
	app.Use(blaze.HTTP2Middleware())
	app.Use(blaze.IPMiddleware()) // Extract client IP information
	app.Use(blaze.Logger())
	app.Use(blaze.Recovery())
	app.Use(blaze.CORS())

	// Routes
	app.GET("/", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"message":   "Hello, HTTP/2 with Blaze!",
			"protocol":  "HTTP/2",
			"client_ip": c.GetUserValue("client_ip"),
			"features": []string{
				"Multiplexing",
				"Server Push",
				"Header Compression",
				"Binary Protocol",
			},
		})
	})

	app.GET("/api/stream/:id", func(c *blaze.Context) error {
		streamID := c.Param("id")

		// Simulate streaming data
		data := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			data[i] = blaze.Map{
				"id":        i + 1,
				"stream_id": streamID,
				"message":   fmt.Sprintf("Stream message %d", i+1),
				"timestamp": time.Now(),
			}
		}

		return c.JSON(blaze.Map{
			"stream_id": streamID,
			"protocol":  "HTTP/2",
			"client_ip": c.GetUserValue("client_ip"),
			"data":      data,
		})
	})

	// Server info endpoint
	app.GET("/api/server-info", func(c *blaze.Context) error {
		return c.JSON(app.GetServerInfo())
	})

	// Performance test endpoint
	app.GET("/api/performance", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"message": "HTTP/2 Performance Test",
			"advantages": []string{
				"Single TCP connection",
				"Request/response multiplexing",
				"Stream prioritization",
				"Server push capabilities",
				"Header compression (HPACK)",
				"Binary framing layer",
			},
			"timestamp": time.Now(),
		})
	})

	log.Printf("🚀 Starting Blaze HTTP/2 server...")
	log.Printf("📍 HTTPS: https://localhost:3443")
	log.Printf("📍 HTTP redirect: http://localhost:3000")
	log.Fatal(app.ListenAndServeGraceful())
}
