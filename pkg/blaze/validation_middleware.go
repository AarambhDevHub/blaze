package blaze

import (
	"net/http"
)

// ValidationMiddleware creates middleware for automatic validation error handling
func ValidationMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Execute next handler
			err := next(c)

			// Check if error is a ValidationErrors
			if validationErr, ok := err.(ValidationErrors); ok {
				return c.Status(http.StatusBadRequest).JSON(Map{
					"success": false,
					"error":   "Validation failed",
					"details": validationErr.Errors,
				})
			}

			// Return original error
			return err
		}
	}
}

// ValidateRequest is a helper function to create route-specific validation middleware
func ValidateRequest(validatorFunc func(*Context) error) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Run custom validation
			if err := validatorFunc(c); err != nil {
				if validationErr, ok := err.(ValidationErrors); ok {
					return c.Status(http.StatusBadRequest).JSON(Map{
						"success": false,
						"error":   "Validation failed",
						"details": validationErr.Errors,
					})
				}
				return c.Status(http.StatusBadRequest).JSON(Map{
					"success": false,
					"error":   err.Error(),
				})
			}

			// Continue to next handler
			return next(c)
		}
	}
}
