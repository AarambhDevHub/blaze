package blaze

import (
	"net/http"
)

// ValidationMiddleware creates middleware for automatic validation error handling
// Intercepts ValidationErrors and converts them to HTTP 400 responses with detailed error information
//
// Middleware Behavior:
//   - Executes next handler in chain
//   - Checks if returned error is ValidationErrors
//   - If validation error: Returns 400 Bad Request with error details
//   - If other error: Passes through to other error handlers
//
// Error Response Format:
//
//	{
//	  "success": false,
//	  "error": "Validation failed",
//	  "details": [
//	    {
//	      "field": "email",
//	      "tag": "email",
//	      "value": "invalid-email",
//	      "message": "email must be a valid email address"
//	    }
//	  ]
//	}
//
// Integration with Validator:
//   - Works with go-playground/validator
//   - Automatically formats validation errors
//   - Provides field-level error messages
//   - Supports custom validation tags
//
// Execution Order:
//   - Should be registered before other error handlers
//   - Applied globally or per-route as needed
//   - Catches validation errors from handlers
//
// Use Cases:
//   - API input validation
//   - Form submission validation
//   - Request body validation
//   - Parameter validation
//
// Returns:
//   - MiddlewareFunc: Validation error handling middleware
//
// Example - Global Validation Middleware:
//
//	app.Use(blaze.ValidationMiddleware())
//
// Example - With Custom Error Handler:
//
//	app.Use(blaze.ValidationMiddleware())
//	app.Use(blaze.ErrorHandlerMiddleware(config))
//
// Example - In Handler:
//
//	type CreateUserRequest struct {
//	    Email    string `json:"email" validate:"required,email"`
//	    Username string `json:"username" validate:"required,min=3,max=20"`
//	    Age      int    `json:"age" validate:"required,gte=18"`
//	}
//
//	func createUser(c *blaze.Context) error {
//	    var req CreateUserRequest
//
//	    // BindJSON parses request body
//	    if err := c.BindJSON(&req); err != nil {
//	        return blaze.ErrBadRequest("Invalid JSON")
//	    }
//
//	    // Validate returns ValidationErrors on failure
//	    if err := c.Validate(&req); err != nil {
//	        return err // Caught by ValidationMiddleware
//	    }
//
//	    // Create user...
//	    return c.JSON(user)
//	}
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
// Allows custom validation logic with automatic error formatting
//
// Validator Function:
//   - Receives context for accessing request data
//   - Returns error (ValidationErrors or other)
//   - Can perform complex validation logic
//   - Can access database or external services
//
// Error Handling:
//   - ValidationErrors: Returns 400 with detailed field errors
//   - Other errors: Returns 400 with generic error message
//   - Nil: Continues to next handler
//
// Parameters:
//   - validatorFunc: Custom validation function
//
// Returns:
//   - MiddlewareFunc: Route-specific validation middleware
//
// Example - Basic Custom Validation:
//
//	validateAge := blaze.ValidateRequest(func(c *blaze.Context) error {
//	    age := c.QueryInt("age")
//	    if age < 18 {
//	        return fmt.Errorf("age must be at least 18")
//	    }
//	    return nil
//	})
//
//	app.GET("/adult-content", handler, validateAge)
//
// Example - Struct Validation:
//
//	validateUser := blaze.ValidateRequest(func(c *blaze.Context) error {
//	    var user User
//	    if err := c.BindJSON(&user); err != nil {
//	        return err
//	    }
//	    return c.Validate(&user)
//	})
//
//	app.POST("/users", createUserHandler, validateUser)
//
// Example - Database Validation:
//
//	validateUniqueEmail := blaze.ValidateRequest(func(c *blaze.Context) error {
//	    var req RegisterRequest
//	    if err := c.BindJSON(&req); err != nil {
//	        return err
//	    }
//
//	    // Check if email exists
//	    exists, err := db.EmailExists(req.Email)
//	    if err != nil {
//	        return err
//	    }
//	    if exists {
//	        return fmt.Errorf("email already registered")
//	    }
//	    return nil
//	})
//
//	app.POST("/register", registerHandler, validateUniqueEmail)
//
// Example - Multi-Field Validation:
//
//	validatePasswords := blaze.ValidateRequest(func(c *blaze.Context) error {
//	    var req ChangePasswordRequest
//	    if err := c.BindJSON(&req); err != nil {
//	        return err
//	    }
//
//	    if req.NewPassword != req.ConfirmPassword {
//	        // Return ValidationErrors for structured errors
//	        return &blaze.ValidationErrors{
//	            Errors: []blaze.ValidationError{
//	                {
//	                    Field: "confirmPassword",
//	                    Message: "passwords do not match",
//	                },
//	            },
//	        }
//	    }
//	    return nil
//	})
//
//	app.POST("/change-password", changePasswordHandler, validatePasswords)
//
// Example - Complex Business Logic:
//
//	validateOrder := blaze.ValidateRequest(func(c *blaze.Context) error {
//	    var order Order
//	    if err := c.BindJSON(&order); err != nil {
//	        return err
//	    }
//
//	    // Validate struct first
//	    if err := c.Validate(&order); err != nil {
//	        return err
//	    }
//
//	    // Check inventory
//	    for _, item := range order.Items {
//	        stock, err := inventory.GetStock(item.ProductID)
//	        if err != nil {
//	            return err
//	        }
//	        if stock < item.Quantity {
//	            return fmt.Errorf("insufficient stock for product %s", item.ProductID)
//	        }
//	    }
//
//	    return nil
//	})
//
//	app.POST("/orders", createOrderHandler, validateOrder)
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
