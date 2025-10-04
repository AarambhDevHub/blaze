//go:build ignore

package main

import (
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Example 1: Serve static files with default configuration
	// Serves all files from ./public directory at /static/* route
	app.Static("/static", "./public")

	// Example 2: Serve static files with custom configuration
	customConfig := blaze.StaticConfig{
		Root:          "./assets",
		Index:         "index.html",
		Browse:        true, // Enable directory listing
		Compress:      true,
		ByteRange:     true,
		CacheDuration: 24 * time.Hour, // Cache for 24 hours
		GenerateETag:  true,
		Exclude:       []string{".git", ".env", ".log"},
		MIMETypes: map[string]string{
			".json": "application/json; charset=utf-8",
		},
		Modify: func(c *blaze.Context) error {
			// Add custom headers
			c.SetHeader("X-Served-By", "Blaze")
			return nil
		},
	}
	app.StaticFS("/assets", customConfig)

	// Example 3: Serve a specific file
	app.File("/favicon.ico", "./public/favicon.ico")
	app.File("/robots.txt", "./public/robots.txt")

	// Example 4: Serve files with custom handler
	app.GET("/download/:filename", func(c *blaze.Context) error {
		filename := c.Param("filename")
		filepath := "./downloads/" + filename

		// Send as download with custom filename
		return c.Download(filepath, "downloaded-"+filename)
	})

	// Example 5: Serve protected static files (with authentication)
	protectedGroup := app.Group("/private")
	protectedGroup.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
		return func(c *blaze.Context) error {
			// Check authentication
			token := c.Header("Authorization")
			if token != "Bearer secret-token" {
				return blaze.ErrUnauthorized("Authentication required")
			}
			return next(c)
		}
	})

	protectedConfig := blaze.DefaultStaticConfig("./private")
	protectedConfig.Browse = false // Disable directory listing for security
	protectedGroup.GET("/*", blaze.StaticFS(protectedConfig))

	// Example 6: Serve single page application (SPA)
	spaConfig := blaze.StaticConfig{
		Root:          "./dist",
		Index:         "index.html",
		Browse:        false,
		CacheDuration: time.Hour,
		NotFoundHandler: func(c *blaze.Context) error {
			// Serve index.html for all routes (SPA routing)
			return c.SendFile("./dist/index.html")
		},
	}
	app.StaticFS("/app", spaConfig)

	// Example 7: Stream large files
	app.GET("/videos/:filename", func(c *blaze.Context) error {
		filename := c.Param("filename")
		filepath := "./videos/" + filename

		// StreamFile supports byte range requests automatically
		return c.StreamFile(filepath)
	})

	// Example 8: Serve files with custom cache control
	app.GET("/images/*", func(c *blaze.Context) error {
		// Set aggressive caching for images
		c.SetHeader("Cache-Control", "public, max-age=31536000, immutable")
		return c.SendFile("./images" + c.Path()[7:]) // Remove "/images" prefix
	})

	// Regular API routes
	app.GET("/", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"message": "Welcome to Blaze Static File Server",
			"routes": []string{
				"/static/* - Public static files",
				"/assets/* - Custom configured assets",
				"/download/:filename - Download files",
				"/private/* - Protected files",
				"/app/* - Single Page Application",
			},
		})
	})

	log.Println("Server starting on http://localhost:3000")
	log.Fatal(app.ListenAndServe())
}
