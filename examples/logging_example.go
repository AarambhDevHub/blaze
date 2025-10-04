//go:build ignore

package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Example 1: Use default logger with middleware
	app.Use(blaze.LoggerMiddleware())

	// Example 2: Configure custom logger
	loggerConfig := blaze.ProductionLoggerConfig()
	loggerConfig.AppName = "my-api"
	loggerConfig.AppVersion = "1.0.0"
	loggerConfig.StaticFields = map[string]interface{}{
		"service": "user-service",
		"region":  "us-east-1",
	}

	customLogger := blaze.NewLogger(loggerConfig)
	blaze.SetDefaultLogger(customLogger)

	// Example 3: File logging
	fileLogger, err := blaze.FileLogger("logs/app.log", blaze.ProductionLoggerConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Example 4: Multi-output logging (console + file)
	file, _ := os.OpenFile("logs/combined.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	multiConfig := blaze.ProductionLoggerConfig()
	multiConfig.Output = blaze.MultiWriter(os.Stdout, file)
	multiLogger := blaze.NewLogger(multiConfig)

	// Example 5: Custom logger middleware with config
	logConfig := blaze.DefaultLoggerMiddlewareConfig()
	logConfig.Logger = multiLogger
	logConfig.LogQueryParams = true
	logConfig.LogHeaders = true
	logConfig.SlowRequestThreshold = 2 * time.Second
	logConfig.CustomFields = func(c *blaze.Context) map[string]interface{} {
		return map[string]interface{}{
			"session_id": c.GetUserValueString("session_id"),
			"tenant_id":  c.GetUserValueString("tenant_id"),
		}
	}

	app.Use(blaze.LoggerMiddlewareWithConfig(logConfig))

	// Example 6: Access log middleware
	app.Use(blaze.AccessLogMiddleware(fileLogger))

	// Example 7: Error logging middleware
	app.Use(blaze.ErrorLogMiddleware(customLogger))

	// Routes with logging

	// Basic route with context logging
	app.GET("/", func(c *blaze.Context) error {
		c.LogInfo("processing home request")
		return c.JSON(blaze.Map{"message": "Hello, Logging!"})
	})

	// Route with structured logging
	app.GET("/users/:id", func(c *blaze.Context) error {
		userID := c.Param("id")

		c.LogInfo("fetching user",
			"user_id", userID,
			"action", "get_user",
		)

		// Simulate user fetch
		time.Sleep(100 * time.Millisecond)

		c.LogDebug("user fetch completed", "user_id", userID)

		return c.JSON(blaze.Map{
			"id":   userID,
			"name": "John Doe",
		})
	})

	// Route with error logging
	app.GET("/error", func(c *blaze.Context) error {
		c.LogWarn("simulating error")
		return blaze.ErrInternalServer("Something went wrong")
	})

	// Route with custom logger
	app.POST("/orders", func(c *blaze.Context) error {
		logger := c.Logger().WithGroup("order")

		logger.Info("creating order")

		// Simulate order processing
		orderID := "ORD-12345"
		amount := 99.99

		logger.Info("order created",
			"order_id", orderID,
			"amount", amount,
			"currency", "USD",
		)

		return c.JSON(blaze.Map{
			"order_id": orderID,
			"status":   "created",
		})
	})

	// Route with slow request
	app.GET("/slow", func(c *blaze.Context) error {
		c.LogInfo("starting slow operation")
		time.Sleep(5 * time.Second)
		c.LogInfo("slow operation completed")
		return c.Text("Done")
	})

	// Route demonstrating different log levels
	app.GET("/logs", func(c *blaze.Context) error {
		c.LogDebug("This is a debug message")
		c.LogInfo("This is an info message")
		c.LogWarn("This is a warning message")
		c.LogError("This is an error message")

		return c.JSON(blaze.Map{"logged": true})
	})

	// Route with database operation logging
	app.GET("/database", func(c *blaze.Context) error {
		dbLogger := c.Logger().WithGroup("database")

		dbLogger.Info("connecting to database")
		time.Sleep(50 * time.Millisecond)

		dbLogger.Info("executing query",
			"table", "users",
			"operation", "SELECT",
		)
		time.Sleep(100 * time.Millisecond)

		dbLogger.Info("query completed",
			"rows_affected", 42,
			"duration_ms", 100,
		)

		return c.JSON(blaze.Map{"users": 42})
	})

	// Route with external API logging
	app.GET("/external", func(c *blaze.Context) error {
		apiLogger := c.Logger().WithGroup("external_api")

		apiLogger.Info("calling external API",
			"url", "https://api.example.com/data",
			"method", "GET",
		)

		// Simulate API call
		time.Sleep(500 * time.Millisecond)

		err := errors.New("connection timeout")
		if err != nil {
			apiLogger.Error("external API call failed",
				"error", err.Error(),
				"retry_count", 3,
			)
			return blaze.ErrExternalAPI("Failed to fetch data", err)
		}

		apiLogger.Info("external API call successful")
		return c.JSON(blaze.Map{"data": "fetched"})
	})

	// Health check (skipped from logging)
	app.GET("/health", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{"status": "healthy"})
	})

	log.Println("Server starting on http://localhost:3000")
	log.Fatal(app.ListenAndServe())
}
