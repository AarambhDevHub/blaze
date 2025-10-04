package blaze

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator wraps go-playground/validator for struct validation
type Validator struct {
	validate *validator.Validate
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (v ValidationErrors) Error() string {
	var messages []string
	for _, err := range v.Errors {
		messages = append(messages, err.Message)
	}
	return strings.Join(messages, "; ")
}

// NewValidator creates a new validator instance
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
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// RegisterValidation registers a custom validation function
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// RegisterStructValidation registers a custom struct validation
func (v *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	v.validate.RegisterStructValidation(fn, types...)
}

// formatErrorMessage formats validation error message
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
func GetValidator() *Validator {
	if defaultValidator == nil {
		defaultValidator = NewValidator()
	}
	return defaultValidator
}

// SetValidator sets a custom global validator
func SetValidator(v *Validator) {
	defaultValidator = v
}
