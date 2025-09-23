//go:build ignore

package main

import (
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	// Create app with HTTP/2 cleartext (h2c) configuration
	config := blaze.DevelopmentConfig()
	config.Host = "127.0.0.1"
	config.Port = 8080
	config.EnableHTTP2 = true
	config.EnableTLS = false // No TLS required for h2c
	config.Development = true

	app := blaze.NewWithConfig(config)

	// Configure HTTP/2 with cleartext (h2c)
	http2Config := blaze.DevelopmentHTTP2Config()
	http2Config.H2C = true // Enable HTTP/2 over cleartext
	http2Config.MaxConcurrentStreams = 500
	http2Config.MaxUploadBufferPerStream = 1024 * 1024 // 1MB
	app.SetHTTP2Config(http2Config)

	// Add middleware
	app.Use(blaze.HTTP2Middleware())
	app.Use(blaze.Logger())
	app.Use(blaze.Recovery())
	app.Use(blaze.CORS())
	app.Use(blaze.IPMiddleware())

	// Routes
	app.GET("/", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"message":   "Hello HTTP/2 Cleartext (h2c)!",
			"protocol":  "HTTP/2 (h2c)",
			"server":    "Blaze Framework",
			"client_ip": c.GetClientIP(),
			"secure":    false,
			"note":      "HTTP/2 without TLS - great for development!",
			"features": []string{
				"Multiplexing (without TLS)",
				"Header Compression",
				"Binary Protocol",
				"Stream Prioritization",
				"No encryption overhead",
			},
		})
	})

	// Concurrent requests demonstration
	app.GET("/api/concurrent/:requests", func(c *blaze.Context) error {
		requestCount, _ := c.ParamInt("requests")
		if requestCount > 50 {
			requestCount = 50 // Safety limit
		}

		responses := make([]map[string]interface{}, requestCount)

		for i := 0; i < requestCount; i++ {
			responses[i] = blaze.Map{
				"request_id": i + 1,
				"message":    "Concurrent HTTP/2 stream",
				"timestamp":  time.Now(),
				"stream":     "Independent stream in same connection",
			}
		}

		return c.JSON(blaze.Map{
			"protocol":           "HTTP/2 (h2c)",
			"concurrent_streams": requestCount,
			"responses":          responses,
			"note":               "All processed concurrently over single connection",
		})
	})

	// HTTP/2 vs HTTP/1.1 comparison
	app.GET("/api/comparison", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"current_protocol": "HTTP/2 (h2c)",
			"comparison": blaze.Map{
				"http1_1": blaze.Map{
					"connections":           "Multiple connections needed",
					"head_of_line_blocking": "Yes - requests block each other",
					"header_compression":    "No",
					"server_push":           "No",
					"multiplexing":          "No",
				},
				"http2_h2c": blaze.Map{
					"connections":           "Single connection for all requests",
					"head_of_line_blocking": "No - streams are independent",
					"header_compression":    "Yes - HPACK",
					"server_push":           "Yes (if supported)",
					"multiplexing":          "Yes - concurrent streams",
				},
			},
			"development_benefits": []string{
				"No TLS certificate management",
				"Easier debugging with tools",
				"Same HTTP/2 features as TLS version",
				"Perfect for local development",
			},
		})
	})

	// Large payload test
	app.POST("/api/large-data", func(c *blaze.Context) error {
		var data interface{}
		if err := c.BindJSON(&data); err != nil {
			return c.Status(400).JSON(blaze.Error("Invalid JSON"))
		}

		// Simulate processing large data
		return c.JSON(blaze.Map{
			"protocol": "HTTP/2 (h2c)",
			"message":  "Large data processed efficiently",
			"received": true,
			"benefits": []string{
				"Stream-based processing",
				"No connection overhead",
				"Efficient bandwidth usage",
				"Concurrent processing possible",
			},
		})
	})

	// Health check
	app.GET("/health", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"status":   "healthy",
			"protocol": "HTTP/2 (h2c)",
			"uptime":   time.Now(),
			"features": []string{
				"Ready for concurrent requests",
				"HTTP/2 multiplexing active",
				"No TLS overhead",
			},
		})
	})

	log.Printf("🚀 Starting HTTP/2 Cleartext (h2c) server...")
	log.Printf("📍 HTTP/2: http://localhost:8080")
	log.Printf("🔗 Try these commands:")
	log.Printf("  curl --http2-prior-knowledge http://localhost:8080/")
	log.Printf("  curl --http2 http://localhost:8080/api/concurrent/10")
	log.Printf("  curl --http2 http://localhost:8080/api/comparison")
	log.Printf("")
	log.Printf("📝 Note: Use --http2-prior-knowledge for direct HTTP/2 connections")
	log.Printf("    or --http2 for HTTP/2 upgrade from HTTP/1.1")

	log.Fatal(app.ListenAndServeGraceful())
}
