//go:build ignore

package main

import (
	"context"
	"log"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Add shutdown-aware middleware
	app.Use(blaze.ShutdownAware())
	app.Use(blaze.GracefulTimeout(30 * time.Second))

	// Basic routes
	app.GET("/", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"message": "Server is running",
			"time":    time.Now(),
		})
	})

	// Long-running endpoint that respects shutdown
	app.GET("/long-task", func(c *blaze.Context) error {
		ctx, cancel := c.WithTimeout(10 * time.Second)
		defer cancel()

		select {
		case <-time.After(5 * time.Second):
			return c.JSON(blaze.Map{
				"message": "Long task completed",
				"time":    time.Now(),
			})
		case <-ctx.Done():
			if c.IsShuttingDown() {
				return c.Status(503).JSON(blaze.Map{
					"error": "Task cancelled due to server shutdown",
				})
			}
			return c.Status(408).JSON(blaze.Map{
				"error": "Task timed out",
			})
		}
	})

	// Health check endpoint
	app.GET("/health", func(c *blaze.Context) error {
		status := "healthy"
		if c.IsShuttingDown() {
			status = "shutting_down"
		}

		return c.JSON(blaze.Map{
			"status": status,
			"time":   time.Now(),
		})
	})

	// Register a cleanup task
	app.RegisterGracefulTask(func(ctx context.Context) error {
		log.Println("🧹 Running cleanup tasks...")

		// Simulate cleanup work
		select {
		case <-time.After(2 * time.Second):
			log.Println("✅ Cleanup completed")
			return nil
		case <-ctx.Done():
			log.Println("⚠️ Cleanup cancelled due to timeout")
			return ctx.Err()
		}
	})

	// Start server with graceful shutdown
	// This will handle SIGINT and SIGTERM automatically
	log.Fatal(app.ListenWithGracefulShutdown())
}
