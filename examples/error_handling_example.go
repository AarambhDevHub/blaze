//go:build ignore

package main

import (
	"errors"
	"log"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Set up error handling (automatically uses development config)
	app.UseErrorHandler(blaze.DevelopmentErrorHandlerConfig())

	// Example 1: Basic HTTP errors
	app.GET("/bad-request", func(c *blaze.Context) error {
		return blaze.ErrBadRequest("Invalid request parameters")
	})

	app.GET("/unauthorized", func(c *blaze.Context) error {
		return blaze.ErrUnauthorized("Authentication required")
	})

	app.GET("/forbidden", func(c *blaze.Context) error {
		return blaze.ErrForbidden("You don't have permission to access this resource")
	})

	app.GET("/not-found", func(c *blaze.Context) error {
		return blaze.ErrNotFound("User not found")
	})

	// Example 2: Errors with details
	app.GET("/validation", func(c *blaze.Context) error {
		validationErrors := map[string]string{
			"email":    "Invalid email format",
			"password": "Password must be at least 8 characters",
		}
		return blaze.ErrValidation("Validation failed", validationErrors)
	})

	// Example 3: Errors with metadata
	app.GET("/with-metadata", func(c *blaze.Context) error {
		return blaze.ErrBadRequest("Invalid user ID").
			WithMetadata("user_id", "12345").
			WithMetadata("attempted_action", "update_profile")
	})

	// Example 4: Database error
	app.GET("/database-error", func(c *blaze.Context) error {
		dbErr := errors.New("connection timeout")
		return blaze.ErrDatabase("Failed to fetch user data", dbErr)
	})

	// Example 5: External API error
	app.GET("/external-api-error", func(c *blaze.Context) error {
		apiErr := errors.New("payment gateway timeout")
		return blaze.ErrExternalAPI("Payment processing failed", apiErr)
	})

	// Example 6: Custom error with stack trace
	app.GET("/with-stack", func(c *blaze.Context) error {
		return blaze.ErrInternalServer("Something went wrong").WithStack(0)
	})

	// Example 7: Panic recovery
	app.GET("/panic", func(c *blaze.Context) error {
		panic("This is a panic!")
	})

	// Example 8: Nested error handling
	app.GET("/nested-error", func(c *blaze.Context) error {
		err := performOperation()
		if err != nil {
			return blaze.ErrInternalServerWithInternal("Operation failed", err)
		}
		return c.JSON(blaze.Map{"success": true})
	})

	// Example 9: Custom error handler for specific route
	customConfig := &blaze.ErrorHandlerConfig{
		IncludeStackTrace:  false,
		LogErrors:          true,
		HideInternalErrors: true,
		CustomHandler: func(c *blaze.Context, err error) error {
			// Custom error handling logic
			return c.Status(500).JSON(blaze.Map{
				"error":   "Custom error handler",
				"message": err.Error(),
			})
		},
	}

	app.GET("/custom-error", func(c *blaze.Context) error {
		err := errors.New("something went wrong")
		return blaze.HandleError(c, err, customConfig)
	})

	// Example 10: Success response
	app.GET("/success", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"success": true,
			"message": "Operation completed successfully",
			"data": map[string]interface{}{
				"user_id": 123,
				"name":    "John Doe",
			},
		})
	})

	log.Fatal(app.ListenAndServe())
}

func performOperation() error {
	// Simulate an operation that might fail
	return errors.New("database connection failed")
}
