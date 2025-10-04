//go:build ignore

package main

import (
	"log"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
	"github.com/go-playground/validator/v10"
)

// User struct with validation tags
type User struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"required,gte=18,lte=100"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	Country  string `json:"country" validate:"required,oneof=US CA UK IN AU"`
}

// LoginRequest with field comparison validation
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegistrationRequest with password confirmation
type RegistrationRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	Age             int    `json:"age" validate:"required,gte=18"`
	Terms           bool   `json:"terms" validate:"required,eq=true"`
}

// Custom validation function for strong password
func strongPasswordValidator(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check for at least one uppercase, one lowercase, one digit, and one special char
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func main() {
	app := blaze.New()

	// Register custom validator
	v := blaze.GetValidator()
	v.RegisterValidation("strongpassword", strongPasswordValidator)

	// Add validation middleware globally
	app.Use(blaze.ValidationMiddleware())

	// Example 1: Basic validation
	app.POST("/users", func(c *blaze.Context) error {
		var user User

		// Bind and validate in one call
		if err := c.BindJSONAndValidate(&user); err != nil {
			return err // Middleware will handle validation errors
		}

		return c.Status(201).JSON(blaze.Map{
			"success": true,
			"message": "User created successfully",
			"user":    user,
		})
	})

	// Example 2: Login with validation
	app.POST("/login", func(c *blaze.Context) error {
		var req LoginRequest

		if err := c.BindJSONAndValidate(&req); err != nil {
			return err
		}

		// Process login...
		return c.JSON(blaze.Map{
			"success": true,
			"message": "Login successful",
			"token":   "sample_jwt_token",
		})
	})

	// Example 3: Registration with custom validation
	app.POST("/register", func(c *blaze.Context) error {
		var req RegistrationRequest

		if err := c.BindJSONAndValidate(&req); err != nil {
			return err
		}

		return c.Status(201).JSON(blaze.Map{
			"success": true,
			"message": "Registration successful",
		})
	})

	// Example 4: Manual validation
	app.GET("/validate/:email", func(c *blaze.Context) error {
		email := c.Param("email")

		// Validate single field
		if err := c.ValidateVar(email, "required,email"); err != nil {
			return c.Status(400).JSON(blaze.Map{
				"success": false,
				"error":   "Invalid email format",
			})
		}

		return c.JSON(blaze.Map{
			"success": true,
			"message": "Valid email",
			"email":   email,
		})
	})

	// Example 5: Route-specific validation middleware using Group
	app.POST("/products", func(c *blaze.Context) error {
		// Custom validation logic
		contentType := c.Header("Content-Type")
		if contentType != "application/json" {
			return blaze.ValidationErrors{
				Errors: []blaze.ValidationError{
					{
						Field:   "Content-Type",
						Tag:     "header",
						Message: "Content-Type must be application/json",
					},
				},
			}
		}

		return c.JSON(blaze.Map{
			"success": true,
			"message": "Product created",
		})
	})

	// Example 6: Nested struct validation
	type Address struct {
		Street  string `json:"street" validate:"required,min=5"`
		City    string `json:"city" validate:"required"`
		ZipCode string `json:"zip_code" validate:"required,numeric,len=5"`
	}

	type UserWithAddress struct {
		Name    string  `json:"name" validate:"required"`
		Email   string  `json:"email" validate:"required,email"`
		Address Address `json:"address" validate:"required,dive"`
	}

	app.POST("/users/with-address", func(c *blaze.Context) error {
		var user UserWithAddress

		if err := c.BindJSONAndValidate(&user); err != nil {
			return err
		}

		return c.Status(201).JSON(blaze.Map{
			"success": true,
			"user":    user,
		})
	})

	log.Fatal(app.ListenAndServe())
}
