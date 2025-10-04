//go:build ignore

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Example 1: Default compression (gzip and deflate)
	app.Use(blaze.Compress())

	// Example 2: Custom compression configuration
	customConfig := blaze.CompressionConfig{
		Level:               blaze.CompressionLevel(6), // Medium compression
		MinLength:           500,                       // Compress responses > 500 bytes
		EnableGzip:          true,
		EnableDeflate:       true,
		EnableBrotli:        false,
		EnableForHTTPS:      false,
		ExcludePaths:        []string{"/api/stream", "/ws"},
		ExcludeExtensions:   []string{".jpg", ".png", ".mp4"},
		IncludeContentTypes: []string{"text/html", "application/json", "text/css"},
	}

	apiGroup := app.Group("/api")
	apiGroup.Use(blaze.CompressWithConfig(customConfig))

	// Example 3: Routes with compression
	app.GET("/", func(c *blaze.Context) error {
		html := `
<!DOCTYPE html>
<html>
<head>
	<title>Compression Test</title>
</head>
<body>
	<h1>Compression Middleware Test</h1>
	<p>` + strings.Repeat("This is a test of the compression middleware. ", 100) + `</p>
</body>
</html>
`
		c.SetHeader("Content-Type", "text/html")
		return c.HTML(html)
	})

	// Example 4: JSON endpoint (will be compressed)
	app.GET("/data", func(c *blaze.Context) error {
		data := make([]map[string]interface{}, 100)
		for i := 0; i < 100; i++ {
			data[i] = blaze.Map{
				"id":          i,
				"name":        fmt.Sprintf("Item %d", i),
				"description": "This is a sample item with a longer description " + strings.Repeat("data ", 10),
				"value":       i * 100,
				"active":      i%2 == 0,
			}
		}
		return c.JSON(blaze.Map{
			"success": true,
			"count":   len(data),
			"data":    data,
		})
	})

	// Example 5: Large text response
	app.GET("/large-text", func(c *blaze.Context) error {
		text := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 1000)
		return c.Text(text)
	})

	// Example 6: Small response (won't be compressed due to size)
	app.GET("/small", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{"message": "small"})
	})

	// Example 7: Binary data (excluded from compression)
	app.GET("/image.png", func(c *blaze.Context) error {
		// Simulate image data
		return c.SendFile("./public/test.png")
	})

	// Example 8: Compression level comparison
	app.GET("/test-compression", func(c *blaze.Context) error {
		largeData := strings.Repeat("Test data for compression analysis. ", 500)

		return c.JSON(blaze.Map{
			"uncompressed_size": len(largeData),
			"data":              largeData,
			"compression_info": blaze.Map{
				"encoding":        c.GetResponseHeader("Content-Encoding"),
				"compressed_size": c.GetResponseHeader("Content-Length"),
				"accept_encoding": c.Header("Accept-Encoding"),
			},
		})
	})

	// Example 9: Group with high compression
	highCompression := app.Group("/high")
	highCompression.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))

	highCompression.GET("/data", func(c *blaze.Context) error {
		data := strings.Repeat("High compression test data. ", 200)
		return c.Text(data)
	})

	// Example 10: Group with fast compression
	fastCompression := app.Group("/fast")
	fastCompression.Use(blaze.CompressWithLevel(blaze.CompressionLevelFastest))

	fastCompression.GET("/data", func(c *blaze.Context) error {
		data := strings.Repeat("Fast compression test data. ", 200)
		return c.Text(data)
	})

	// Example 11: Selective compression by content type
	app.GET("/json-only", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"compressed": true,
			"data":       strings.Repeat("JSON data ", 100),
		})
	})

	// Example 12: Compression stats endpoint
	app.GET("/compression-info", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"enabled":    true,
			"algorithms": []string{"gzip", "deflate"},
			"min_length": 1024,
			"excluded_types": []string{
				"image/*", "video/*", "audio/*",
				"application/zip", "application/gzip",
			},
			"usage": "Add 'Accept-Encoding: gzip' header to your requests",
		})
	})

	log.Println("Server starting on http://localhost:3000")
	log.Println("Test with: curl -H 'Accept-Encoding: gzip' http://localhost:3000/data -i")
	log.Fatal(app.ListenAndServe())
}
