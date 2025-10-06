package blaze

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator for struct validation
// Provides integration between Blaze and the popular validator package
//
// Validator Features:
//   - Struct field validation with tags
//   - Custom validation rules
//   - Cross-field validation
//   - Nested struct validation
//   - Array and slice validation
//   - Custom error messages
//
// Validation Tags:
//   - required: Field must have a non-zero value
//   - email: Must be valid email format
//   - min/max: String length or numeric bounds
//   - len: Exact length requirement
//   - url/uri: Valid URL/URI format
//   - uuid: Valid UUID format
//   - oneof: Value must be one of specified options
//   - And many more...
//
// Thread Safety:
//   - Safe for concurrent use
//   - Validator instance can be shared
type Validator struct {
	// validate is the underlying validator instance
	validate *validator.Validate
}

// ValidationError represents a single validation error
// Provides detailed information about what failed validation
//
// Error Information:
//   - Field: Name of the field that failed
//   - Tag: Validation tag that failed
//   - Value: The actual value that was invalid
//   - Message: Human-readable error message
type ValidationError struct {
	Field   string      `json:"field"`           // Field name (JSON tag if present)
	Tag     string      `json:"tag"`             // Validation tag that failed
	Value   interface{} `json:"value,omitempty"` // The invalid value
	Message string      `json:"message"`         // Human-readable error message
}

// ValidationErrors represents multiple validation errors
// Container for all validation failures in a struct
//
// Error Response Format:
//
//	{
//	  "errors": [
//	    {
//	      "field": "email",
//	      "tag": "email",
//	      "value": "invalid-email",
//	      "message": "email must be a valid email address"
//	    },
//	    {
//	      "field": "age",
//	      "tag": "gte",
//	      "value": 15,
//	      "message": "age must be greater than or equal to 18"
//	    }
//	  ]
//	}
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
// Returns concatenated error messages from all validation failures
//
// Returns:
//   - string: Combined error messages separated by semicolons
func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// NewValidator creates a new validator instance
// Configures validator with JSON field names and custom settings
//
// Configuration:
//   - Uses JSON field names in error messages
//   - Falls back to struct field name if no JSON tag
//   - Ready for custom validation registration
//
// Returns:
//   - *Validator: Configured validator instance
//
// Example:
//
//	validator := blaze.NewValidator()
//	validator.RegisterValidation("password", passwordValidator)
func NewValidator() *Validator {
	v := validator.New()

	// Use JSON field names in error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	return &Validator{
		validate: v,
	}
}

// ValidateStruct validates a struct and returns formatted errors
// Performs validation and converts errors to user-friendly format
//
// Validation Process:
//  1. Validate struct using tags
//  2. If validation passes, return nil
//  3. If validation fails, convert to ValidationErrors
//  4. Format error messages for each field
//
// Parameters:
//   - s: Struct to validate (must be struct or pointer to struct)
//
// Returns:
//   - error: ValidationErrors with detailed field errors, or nil if valid
//
// Example - Basic Validation:
//
//	type User struct {
//	    Email    string `json:"email" validate:"required,email"`
//	    Username string `json:"username" validate:"required,min=3,max=20"`
//	    Age      int    `json:"age" validate:"required,gte=18"`
//	}
//
//	user := User{Email: "invalid", Username: "ab", Age: 15}
//	validator := blaze.GetValidator()
//
//	if err := validator.ValidateStruct(&user); err != nil {
//	    // Returns ValidationErrors with 3 field errors
//	    return c.Status(400).JSON(err)
//	}
//
// Example - Nested Struct Validation:
//
//	type Address struct {
//	    Street string `json:"street" validate:"required"`
//	    City   string `json:"city" validate:"required"`
//	    Zip    string `json:"zip" validate:"required,len=5"`
//	}
//
//	type User struct {
//	    Name    string  `json:"name" validate:"required"`
//	    Address Address `json:"address" validate:"required"`
//	}
//
//	user := User{Name: "John"}
//	if err := validator.ValidateStruct(&user); err != nil {
//	    // Validates nested Address struct
//	}
//
// Example - Array Validation:
//
//	type Post struct {
//	    Title string   `json:"title" validate:"required"`
//	    Tags  []string `json:"tags" validate:"required,min=1,dive,min=3"`
//	}
//
//	post := Post{Title: "Hello", Tags: []string{"ab"}}
//	if err := validator.ValidateStruct(&post); err != nil {
//	    // Validates array has at least one item
//	    // Validates each tag is at least 3 characters
//	}
func (v *Validator) ValidateStruct(s interface{}) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	// Convert validator errors to our custom format
	validationErrors := ValidationErrors{
		Errors: make([]ValidationError, 0),
	}

	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			validationError := ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fieldError.Value(),
				Message: v.formatErrorMessage(fieldError),
			}
			validationErrors.Errors = append(validationErrors.Errors, validationError)
		}
	}

	return validationErrors
}

// ValidateVar validates a single variable
// Useful for validating individual values outside of structs
//
// Parameters:
//   - field: Value to validate
//   - tag: Validation tag string
//
// Returns:
//   - error: Validation error or nil if valid
//
// Example - Email Validation:
//
//	email := "test@example.com"
//	err := validator.ValidateVar(email, "required,email")
//
// Example - Number Range:
//
//	age := 25
//	err := validator.ValidateVar(age, "gte=18,lte=100")
//
// Example - String Length:
//
//	password := "secret123"
//	err := validator.ValidateVar(password, "required,min=8,max=50")
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// RegisterValidation registers a custom validation function
// Allows defining custom validation rules beyond built-in tags
//
// Validator Function Signature:
//
//	func(fl validator.FieldLevel) bool
//	- Return true if valid
//	- Return false if invalid
//	- Access field value via fl.Field()
//
// Parameters:
//   - tag: Custom validation tag name
//   - fn: Validation function
//
// Returns:
//   - error: Registration error or nil
//
// Example - Password Strength:
//
//	validator.RegisterValidation("password", func(fl validator.FieldLevel) bool {
//	    password := fl.Field().String()
//	    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
//	    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
//	    hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
//	    return hasUpper && hasLower && hasDigit && len(password) >= 8
//	})
//
//	type User struct {
//	    Password string `validate:"password"`
//	}
//
// Example - Unique Username (with database):
//
//	validator.RegisterValidation("unique_username", func(fl validator.FieldLevel) bool {
//	    username := fl.Field().String()
//	    exists, _ := db.UsernameExists(username)
//	    return !exists
//	})
//
// Example - Business Hours:
//
//	validator.RegisterValidation("business_hours", func(fl validator.FieldLevel) bool {
//	    hour := fl.Field().Int()
//	    return hour >= 9 && hour <= 17
//	})
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// RegisterStructValidation registers a custom struct validation function
// Allows validating relationships between multiple fields
//
// Use Cases:
//   - Cross-field validation
//   - Complex business rules
//   - Conditional validation
//
// Parameters:
//   - fn: Struct-level validation function
//   - types: Struct types to apply validation to
//
// Example - Password Confirmation:
//
//	validator.RegisterStructValidation(func(sl validator.StructLevel) {
//	    user := sl.Current().Interface().(User)
//	    if user.Password != user.ConfirmPassword {
//	        sl.ReportError(user.ConfirmPassword, "confirm_password", "ConfirmPassword", "eqfield", "Password")
//	    }
//	}, User{})
//
// Example - Date Range:
//
//	validator.RegisterStructValidation(func(sl validator.StructLevel) {
//	    event := sl.Current().Interface().(Event)
//	    if event.EndDate.Before(event.StartDate) {
//	        sl.ReportError(event.EndDate, "end_date", "EndDate", "gtfield", "StartDate")
//	    }
//	}, Event{})
func (v *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	v.validate.RegisterStructValidation(fn, types...)
}

// formatErrorMessage formats validation error message
// Converts validation tag to human-readable message
//
// Message Templates:
//   - required: "field is required"
//   - email: "field must be a valid email address"
//   - min: "field must be at least N characters long"
//   - max: "field must be at most N characters long"
//   - And many more...
//
// Parameters:
//   - fe: Field error from validator
//
// Returns:
//   - string: Formatted error message
func (v *Validator) formatErrorMessage(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, param)
	case "eq":
		return fmt.Sprintf("%s must be equal to %s", field, param)
	case "ne":
		return fmt.Sprintf("%s must not be equal to %s", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "numeric":
		return fmt.Sprintf("%s must be a number", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uri":
		return fmt.Sprintf("%s must be a valid URI", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, param)
	case "eqfield":
		return fmt.Sprintf("%s must equal %s", field, param)
	case "nefield":
		return fmt.Sprintf("%s must not equal %s", field, param)
	case "gtfield":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "gtefield":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "ltfield":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "ltefield":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "containsany":
		return fmt.Sprintf("%s must contain at least one of the following characters: %s", field, param)
	case "excludes":
		return fmt.Sprintf("%s must not contain %s", field, param)
	case "datetime":
		return fmt.Sprintf("%s must be a valid datetime in format %s", field, param)
	default:
		return fmt.Sprintf("%s failed validation on tag %s", field, tag)
	}
}

// Global validator instance
var defaultValidator *Validator

// GetValidator returns the global validator instance
// Creates default validator if not initialized
//
// Returns:
//   - *Validator: Global validator instance
//
// Example:
//
//	validator := blaze.GetValidator()
//	err := validator.ValidateStruct(&user)
func GetValidator() *Validator {
	if defaultValidator == nil {
		defaultValidator = NewValidator()
	}
	return defaultValidator
}

// SetValidator sets a custom global validator
// Allows replacing default validator with custom instance
//
// Parameters:
//   - v: Custom validator instance
//
// Example:
//
//	customValidator := blaze.NewValidator()
//	customValidator.RegisterValidation("custom", customFunc)
//	blaze.SetValidator(customValidator)
func SetValidator(v *Validator) {
	defaultValidator = v
}
