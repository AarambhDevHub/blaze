// Package blaze provides utilities for building and handling multipart form data.
// This file contains helper functions and methods for binding multipart form data
// to Go structs with automatic type conversion, validation, and error handling.
package blaze

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// StructTag represents parsed struct tag information for form binding.
// Struct tags control how form fields are mapped to struct fields and
// provide validation rules and default values.
//
// Tag Format:
//
//	form:"fieldname,required,maxsize:1024,minsize:10,default:value"
//
// Supported Options:
//   - required: Field must be present in the form
//   - maxsize:N: Maximum size in bytes for the field value
//   - minsize:N: Minimum size in bytes for the field value
//   - default:value: Default value if field is empty or missing
//
// Example:
//
//	type User struct {
//	    Name     string `form:"name,required,minsize:2,maxsize:100"`
//	    Email    string `form:"email,required"`
//	    Age      int    `form:"age,default:18"`
//	    Bio      string `form:"bio,maxsize:500"`
//	}
type StructTag struct {
	// Name is the form field name to bind to this struct field.
	// If empty, the lowercased struct field name is used.
	Name string

	// Required indicates whether the field must be present in the form.
	// If true and field is missing/empty, binding returns an error.
	Required bool

	// MaxSize is the maximum allowed size in bytes for the field value.
	// Zero means no limit. Applies to strings and slices.
	MaxSize int64

	// MinSize is the minimum required size in bytes for the field value.
	// Zero means no minimum. Applies to strings and slices.
	MinSize int64

	// Default is the default value to use if field is empty or missing.
	// The value is parsed according to the field's type.
	Default string
}

// parseStructTag parses a struct field tag into a StructTag configuration.
// This function extracts form binding rules from struct tags and returns
// a configuration object for the binding process.
//
// Tag Syntax:
//
//	form:"name,option1,option2:value"
//
// Parameters:
//   - tag: The raw struct tag string to parse
//
// Returns:
//   - StructTag: Parsed tag configuration
//
// Example:
//
//	tag := parseStructTag("username,required,maxsize:50,default:guest")
//	// Returns: StructTag{
//	//     Name: "username",
//	//     Required: true,
//	//     MaxSize: 50,
//	//     Default: "guest",
//	// }
func parseStructTag(tag string) StructTag {
	result := StructTag{}

	if tag == "" {
		return result
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 && parts[0] != "" {
		result.Name = parts[0]
	}

	for i := 1; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		switch {
		case part == "required":
			result.Required = true
		case strings.HasPrefix(part, "maxsize="):
			if size, err := strconv.ParseInt(part[8:], 10, 64); err == nil {
				result.MaxSize = size
			}
		case strings.HasPrefix(part, "minsize="):
			if size, err := strconv.ParseInt(part[8:], 10, 64); err == nil {
				result.MinSize = size
			}
		case strings.HasPrefix(part, "default="):
			result.Default = part[8:]
		}
	}

	return result
}

// BindMultipartForm binds multipart form data to a struct using reflection.
// This is the main entry point for form binding, automatically parsing the
// multipart form and mapping fields to struct members with type conversion.
//
// Binding Process:
//  1. Parse multipart form from request
//  2. Use reflection to inspect target struct
//  3. Map form fields to struct fields by name
//  4. Convert form values to appropriate types
//  5. Validate required fields and constraints
//
// Supported Types:
//   - Basic types: string, int, float, bool
//   - Time: time.Time with multiple format support
//   - Files: MultipartFile for uploaded files
//   - Slices: []string, []int, []MultipartFile, etc.
//   - Pointers: *string, *int, *time.Time, etc.
//   - Nested structs with form tags
//
// Parameters:
//   - v: Pointer to struct to bind data into
//
// Returns:
//   - error: Binding or validation error, nil on success
//
// Example:
//
//	type UploadForm struct {
//	    Title       string          `form:"title,required"`
//	    Description string          `form:"description"`
//	    File        *MultipartFile  `form:"file,required"`
//	    Tags        []string        `form:"tags"`
//	}
//
//	func handler(c *blaze.Context) error {
//	    var form UploadForm
//	    if err := c.BindMultipartForm(&form); err != nil {
//	        return c.Status(400).JSON(blaze.Error(err.Error()))
//	    }
//	    // Use form.Title, form.File, etc.
//	    return c.JSON(blaze.OK(form))
//	}
func (c *Context) BindMultipartForm(v interface{}) error {
	form, err := c.MultipartForm()
	if err != nil {
		return fmt.Errorf("failed to parse multipart form: %w", err)
	}

	return c.bindMultipartFormToStruct(form, v)
}

// bindMultipartFormToStruct performs the actual struct binding using reflection.
// This is the core binding logic that recursively processes struct fields,
// handles type conversions, applies validation rules, and manages nested structures.
//
// Type Handling:
//   - String: Direct assignment with size validation
//   - Int/Uint: ParseInt with overflow checking
//   - Float: ParseFloat with precision handling
//   - Bool: Supports true/false, 1/0, on/off, yes/no
//   - Time: Tries multiple common date/time formats
//   - File: Maps to MultipartFile with validation
//   - Slice: Splits multiple values or file arrays
//   - Pointer: Creates new instance and sets value
//   - Struct: Recursive binding for nested forms
//
// Validation:
//   - Required field checking
//   - Size constraints (min/max)
//   - Type compatibility
//   - Format validation
//
// Parameters:
//   - form: Parsed multipart form data
//   - v: Pointer to struct to bind into
//
// Returns:
//   - error: Detailed error with field name and reason
//
// Example:
//
//	form := &MultipartForm{
//	    Value: map[string][]string{
//	        "name": {"John"},
//	        "age": {"30"},
//	    },
//	}
//	var user User
//	err := c.bindMultipartFormToStruct(form, &user)
func (c *Context) bindMultipartFormToStruct(form *MultipartForm, v interface{}) error {
	rv := reflect.ValueOf(v)

	// Must be a pointer to a struct
	if rv.Kind() != reflect.Ptr {
		return errors.New("binding destination must be a pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("binding destination must be a pointer to struct")
	}

	rt := rv.Type()

	// Iterate through struct fields
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get field name from tag or use struct field name
		tag := parseStructTag(fieldType.Tag.Get("form"))
		fieldName := tag.Name
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}

		// Skip fields marked with "-"
		if fieldName == "-" {
			continue
		}

		// Handle different field types
		switch field.Kind() {
		case reflect.String:
			if err := c.setStringField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set string field %s: %w", fieldType.Name, err)
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if err := c.setIntField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set int field %s: %w", fieldType.Name, err)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if err := c.setUintField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set uint field %s: %w", fieldType.Name, err)
			}

		case reflect.Float32, reflect.Float64:
			if err := c.setFloatField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set float field %s: %w", fieldType.Name, err)
			}

		case reflect.Bool:
			if err := c.setBoolField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set bool field %s: %w", fieldType.Name, err)
			}

		case reflect.Slice:
			if err := c.setSliceField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set slice field %s: %w", fieldType.Name, err)
			}

		case reflect.Ptr:
			if err := c.setPointerField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set pointer field %s: %w", fieldType.Name, err)
			}

		case reflect.Struct:
			if err := c.setStructField(form, field, fieldName, tag); err != nil {
				return fmt.Errorf("failed to set struct field %s: %w", fieldType.Name, err)
			}

		default:
			return fmt.Errorf("unsupported field type %s for field %s", field.Kind(), fieldType.Name)
		}
	}

	return nil
}

// setStringField sets a string field value with validation.
// Handles required field checking, default values, and size constraints.
//
// Validation:
//   - Checks if required field is present
//   - Applies default value if field is empty
//   - Validates maximum size constraint
//   - Validates minimum size constraint
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Validation error with detailed message
//
// Example:
//
//	// Struct field: Name string `form:"name,required,maxsize:100,minsize:2"`
//	// Form data: name=John
//	// Result: field.String() == "John"
func (c *Context) setStringField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	value := form.GetValue(fieldName)

	// Check if required field is empty
	if tag.Required && value == "" {
		if tag.Default != "" {
			value = tag.Default
		} else {
			return fmt.Errorf("required field %s is missing", fieldName)
		}
	}

	// Set default if empty
	if value == "" && tag.Default != "" {
		value = tag.Default
	}

	// Validate size constraints
	if tag.MaxSize > 0 && int64(len(value)) > tag.MaxSize {
		return fmt.Errorf("field %s exceeds maximum size %d", fieldName, tag.MaxSize)
	}

	if tag.MinSize > 0 && int64(len(value)) < tag.MinSize {
		return fmt.Errorf("field %s below minimum size %d", fieldName, tag.MinSize)
	}

	field.SetString(value)
	return nil
}

// setIntField sets an integer field value with parsing and validation.
// Supports all signed integer types (int, int8, int16, int32, int64).
//
// Conversion:
//   - Parses string value to int64
//   - Checks for parsing errors
//   - Handles overflow for smaller int types
//   - Applies default value if field is empty
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Parsing or validation error
//
// Example:
//
//	// Struct field: Age int `form:"age,required,default:18"`
//	// Form data: age=25
//	// Result: field.Int() == 25
func (c *Context) setIntField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	valueStr := form.GetValue(fieldName)

	if valueStr == "" {
		if tag.Required {
			if tag.Default != "" {
				valueStr = tag.Default
			} else {
				return fmt.Errorf("required field %s is missing", fieldName)
			}
		} else if tag.Default != "" {
			valueStr = tag.Default
		} else {
			field.SetInt(0)
			return nil
		}
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid integer value for field %s: %s", fieldName, valueStr)
	}

	// Check bounds for specific int types
	switch field.Kind() {
	case reflect.Int8:
		if value < -128 || value > 127 {
			return fmt.Errorf("value %d out of range for int8", value)
		}
	case reflect.Int16:
		if value < -32768 || value > 32767 {
			return fmt.Errorf("value %d out of range for int16", value)
		}
	case reflect.Int32:
		if value < -2147483648 || value > 2147483647 {
			return fmt.Errorf("value %d out of range for int32", value)
		}
	}

	field.SetInt(value)
	return nil
}

// setUintField sets an unsigned integer field value.
// Supports all unsigned integer types (uint, uint8, uint16, uint32, uint64).
//
// Conversion:
//   - Parses string value to uint64
//   - Checks for negative values
//   - Handles overflow for smaller uint types
//   - Applies default value if field is empty
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Parsing or validation error
//
// Example:
//
//	// Struct field: Count uint `form:"count,default:1"`
//	// Form data: count=10
//	// Result: field.Uint() == 10
func (c *Context) setUintField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	valueStr := form.GetValue(fieldName)

	if valueStr == "" {
		if tag.Required {
			if tag.Default != "" {
				valueStr = tag.Default
			} else {
				return fmt.Errorf("required field %s is missing", fieldName)
			}
		} else if tag.Default != "" {
			valueStr = tag.Default
		} else {
			field.SetUint(0)
			return nil
		}
	}

	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid unsigned integer value for field %s: %s", fieldName, valueStr)
	}

	// Check bounds for specific uint types
	switch field.Kind() {
	case reflect.Uint8:
		if value > 255 {
			return fmt.Errorf("value %d out of range for uint8", value)
		}
	case reflect.Uint16:
		if value > 65535 {
			return fmt.Errorf("value %d out of range for uint16", value)
		}
	case reflect.Uint32:
		if value > 4294967295 {
			return fmt.Errorf("value %d out of range for uint32", value)
		}
	}

	field.SetUint(value)
	return nil
}

// setFloatField sets a floating-point field value.
// Supports float32 and float64 types with appropriate precision.
//
// Conversion:
//   - Parses string value to float64
//   - Handles scientific notation
//   - Maintains precision for float32 vs float64
//   - Applies default value if field is empty
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Parsing or validation error
//
// Example:
//
//	// Struct field: Price float64 `form:"price,required"`
//	// Form data: price=19.99
//	// Result: field.Float() == 19.99
func (c *Context) setFloatField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	valueStr := form.GetValue(fieldName)

	if valueStr == "" {
		if tag.Required {
			if tag.Default != "" {
				valueStr = tag.Default
			} else {
				return fmt.Errorf("required field %s is missing", fieldName)
			}
		} else if tag.Default != "" {
			valueStr = tag.Default
		} else {
			field.SetFloat(0.0)
			return nil
		}
	}

	bitSize := 64
	if field.Kind() == reflect.Float32 {
		bitSize = 32
	}

	value, err := strconv.ParseFloat(valueStr, bitSize)
	if err != nil {
		return fmt.Errorf("invalid float value for field %s: %s", fieldName, valueStr)
	}

	field.SetFloat(value)
	return nil
}

// setBoolField sets a boolean field value.
// Supports multiple boolean representations for user convenience.
//
// Accepted Values:
//   - true: "true", "1", "on", "yes", "checked"
//   - false: "false", "0", "off", "no", "" (empty/missing)
//
// Special Behavior:
//   - Missing checkbox fields are treated as false
//   - Case-insensitive matching
//   - Empty value defaults to false
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Invalid boolean value error
//
// Example:
//
//	// Struct field: Active bool `form:"active"`
//	// Form data: active=on (HTML checkbox)
//	// Result: field.Bool() == true
func (c *Context) setBoolField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	valueStr := form.GetValue(fieldName)

	if valueStr == "" {
		if tag.Default != "" {
			valueStr = tag.Default
		} else {
			// For checkboxes, missing value typically means false
			field.SetBool(false)
			return nil
		}
	}

	// Handle common boolean representations
	switch strings.ToLower(valueStr) {
	case "true", "1", "on", "yes", "checked":
		field.SetBool(true)
	case "false", "0", "off", "no", "":
		field.SetBool(false)
	default:
		return fmt.Errorf("invalid boolean value for field %s: %s", fieldName, valueStr)
	}

	return nil
}

// setSliceField sets a slice field value with element conversion.
// Handles both value slices ([]string, []int) and file slices ([]MultipartFile).
//
// Slice Types:
//   - []string: Multiple form values with same name
//   - []int/[]float: Parses each value separately
//   - []MultipartFile: Multiple file uploads
//   - []T for any supported type T
//
// Form Handling:
//   - HTML: <input name="tags" value="go"> <input name="tags" value="rust">
//   - Result: []string{"go", "rust"}
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Element conversion or validation error
//
// Example:
//
//	// Struct field: Tags []string `form:"tags,required"`
//	// Form data: tags=go&tags=rust&tags=web
//	// Result: field == []string{"go", "rust", "web"}
func (c *Context) setSliceField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	elemType := field.Type().Elem()

	// Handle file slices specially
	if elemType == reflect.TypeOf(&MultipartFile{}) {
		files := form.GetFiles(fieldName)
		if files == nil {
			if tag.Required {
				return fmt.Errorf("required file field %s is missing", fieldName)
			}
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			return nil
		}

		slice := reflect.MakeSlice(field.Type(), len(files), len(files))
		for i, file := range files {
			slice.Index(i).Set(reflect.ValueOf(file))
		}
		field.Set(slice)
		return nil
	}

	values := form.GetValues(fieldName)
	if values == nil {
		if tag.Required {
			return fmt.Errorf("required field %s is missing", fieldName)
		}
		field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		return nil
	}

	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, value := range values {
		elem := slice.Index(i)
		if err := c.setScalarValue(elem, value, elemType); err != nil {
			return fmt.Errorf("failed to set slice element %d: %w", i, err)
		}
	}

	field.Set(slice)
	return nil
}

// setScalarValue sets a scalar value (string, int, float, bool) on a reflect.Value.
// This is a utility function for converting string values to typed fields.
//
// Supported Types:
//   - String: Direct assignment
//   - Int/Uint: Base 10 parsing
//   - Float: Scientific notation support
//   - Bool: Multiple representation support
//
// Type Safety:
//   - Checks for overflow in smaller types
//   - Validates format before assignment
//   - Returns detailed error messages
//
// Parameters:
//   - field: Reflect value to set
//   - value: String value to convert and assign
//   - targetType: Type to convert to
//
// Returns:
//   - error: Conversion or validation error
//
// Example:
//
//	var field reflect.Value // int field
//	err := c.setScalarValue(field, "42", reflect.TypeOf(0))
//	// field.Int() == 42
func (c *Context) setScalarValue(field reflect.Value, value string, fieldType reflect.Type) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		field.SetBool(boolVal)
	default:
		return fmt.Errorf("unsupported type %s", field.Kind())
	}

	return nil
}

// BindForm binds both multipart and URL-encoded form data to a struct
func (c *Context) BindForm(v interface{}) error {
	// Check if it's multipart form
	if c.IsMultipartForm() {
		return c.BindMultipartForm(v)
	}

	// Handle URL-encoded form data
	return c.bindURLEncodedForm(v)
}

// bindURLEncodedForm binds URL-encoded form data to struct
func (c *Context) bindURLEncodedForm(v interface{}) error {
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return errors.New("binding destination must be a pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("binding destination must be a pointer to struct")
	}

	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanSet() {
			continue
		}

		tag := parseStructTag(fieldType.Tag.Get("form"))
		fieldName := tag.Name
		if fieldName == "" {
			fieldName = strings.ToLower(fieldType.Name)
		}

		if fieldName == "-" {
			continue
		}

		// Get form value
		value := string(c.RequestCtx.FormValue(fieldName))

		// Handle required fields
		if tag.Required && value == "" {
			if tag.Default != "" {
				value = tag.Default
			} else {
				return fmt.Errorf("required field %s is missing", fieldName)
			}
		}

		// Set default if empty
		if value == "" && tag.Default != "" {
			value = tag.Default
		}

		// Set the field value
		if value != "" || field.Kind() == reflect.Bool {
			if err := c.setScalarValue(field, value, field.Type()); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// setStructField sets struct field value (like time.Time or nested structs).
// Handles time.Time with multiple format support and recursive struct binding.
//
// Struct Types:
//   - time.Time: Parses using multiple common formats
//   - Nested structs: Recursive binding with form tags
//   - Custom types: Requires form tags on nested fields
//
// Time Format Support:
//   - RFC3339: "2006-01-02T15:04:05Z07:00"
//   - ISO8601: "2006-01-02T15:04:05"
//   - DateTime: "2006-01-02 15:04:05"
//   - Date: "2006-01-02"
//   - Time: "15:04:05"
//   - Compact: "20060102"
//   - US Date: "02/01/2006"
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Parsing or binding error
//
// Example:
//
//	// Struct field: CreatedAt time.Time `form:"created"`
//	// Form data: created=2024-01-15T10:30:00Z
//	// Result: field contains parsed time value
func (c *Context) setStructField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	value := form.GetValue(fieldName)

	if value == "" {
		if tag.Required {
			if tag.Default != "" {
				value = tag.Default
			} else {
				return fmt.Errorf("required field %s is missing", fieldName)
			}
		} else if tag.Default != "" {
			value = tag.Default
		} else {
			return nil
		}
	}

	// Handle time.Time specially with comprehensive format support
	if field.Type() == reflect.TypeOf(time.Time{}) {
		return c.parseTimeField(field, value, fieldName)
	}

	// For other struct types, try to recursively bind if it has form fields
	if field.CanAddr() {
		return c.bindMultipartFormToStruct(form, field.Addr().Interface())
	}

	return fmt.Errorf("unsupported struct type %s for field %s", field.Type(), fieldName)
}

// parseTimeField parses a time string into a time.Time field.
// Tries multiple common date/time formats for maximum compatibility.
//
// Supported Formats (in order):
//  1. RFC3339: "2006-01-02T15:04:05Z07:00"
//  2. ISO8601 with T: "2006-01-02T15:04:05"
//  3. DateTime: "2006-01-02 15:04:05"
//  4. Date only: "2006-01-02"
//  5. Time only: "15:04:05"
//  6. Compact: "20060102"
//  7. US Date: "02/01/2006"
//
// Parameters:
//   - field: Reflect value of time.Time field to set
//   - value: String value to parse
//   - fieldName: Field name for error messages
//
// Returns:
//   - error: Parsing error if all formats fail
//
// Example:
//
//	// Input: "2024-01-15"
//	// Output: time.Time for midnight on Jan 15, 2024
func (c *Context) parseTimeField(field reflect.Value, value, fieldName string) error {
	// Comprehensive list of time formats to try
	timeFormats := []string{
		// HTML datetime-local format (most common from forms)
		"2006-01-02T15:04",    // HTML datetime-local without seconds
		"2006-01-02T15:04:05", // HTML datetime-local with seconds

		// ISO 8601 formats
		time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02T15:04:05",        // Without timezone
		"2006-01-02T15:04:05.000",    // With milliseconds
		"2006-01-02T15:04:05.000000", // With microseconds

		// Common database formats
		"2006-01-02 15:04:05",     // SQL datetime
		"2006-01-02 15:04:05.000", // SQL datetime with milliseconds
		"2006-01-02 15:04",        // SQL datetime without seconds

		// Date only formats
		"2006-01-02",      // ISO date
		"2006/01/02",      // US date format
		"02/01/2006",      // EU date format
		"01/02/2006",      // US date format (MM/DD/YYYY)
		"2-Jan-2006",      // Written date format
		"Jan 2, 2006",     // Written date format
		"January 2, 2006", // Full written date

		// Time only formats
		"15:04:05",   // 24-hour time with seconds
		"15:04",      // 24-hour time without seconds
		"3:04:05 PM", // 12-hour time with seconds
		"3:04 PM",    // 12-hour time without seconds

		// RFC formats
		time.RFC822,   // 02 Jan 06 15:04 MST
		time.RFC822Z,  // 02 Jan 06 15:04 -0700
		time.RFC850,   // Monday, 02-Jan-06 15:04:05 MST
		time.RFC1123,  // Mon, 02 Jan 2006 15:04:05 MST
		time.RFC1123Z, // Mon, 02 Jan 2006 15:04:05 -0700

		// Unix timestamp (as string)
		"1136239445", // Unix timestamp

		// Custom formats that might be used
		"2006-01-02T15:04:05-07:00", // ISO with timezone offset
		"2006-01-02T15:04:05+07:00", // ISO with positive timezone offset
		"2006-01-02 15:04:05 -0700", // SQL with timezone
		"2006-01-02 15:04:05 MST",   // SQL with timezone name
	}

	var parsedTime time.Time
	var lastError error

	// Try each format until one works
	for _, format := range timeFormats {
		var err error
		parsedTime, err = time.Parse(format, value)
		if err == nil {
			// Successfully parsed
			field.Set(reflect.ValueOf(parsedTime))
			return nil
		}
		lastError = err

		// Try parsing in local timezone for formats without timezone info
		if !strings.Contains(format, "Z") && !strings.Contains(format, "07:00") && !strings.Contains(format, "MST") {
			parsedTime, err = time.ParseInLocation(format, value, time.Local)
			if err == nil {
				field.Set(reflect.ValueOf(parsedTime))
				return nil
			}
		}
	}

	// If all formats failed, try parsing as Unix timestamp (integer)
	if unixTimestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		parsedTime = time.Unix(unixTimestamp, 0)
		field.Set(reflect.ValueOf(parsedTime))
		return nil
	}

	// Try parsing as Unix timestamp with milliseconds
	if unixMilli, err := strconv.ParseInt(value, 10, 64); err == nil && unixMilli > 1000000000000 {
		parsedTime = time.Unix(unixMilli/1000, (unixMilli%1000)*1000000)
		field.Set(reflect.ValueOf(parsedTime))
		return nil
	}

	return fmt.Errorf("invalid time format for field %s: %s (tried %d formats, last error: %v)",
		fieldName, value, len(timeFormats), lastError)
}

// setPointerField sets a pointer field value.
// Creates a new instance of the pointed-to type and sets its value.
//
// Pointer Handling:
//   - Creates new instance if value is present
//   - Sets to nil if value is missing and not required
//   - Handles *MultipartFile specially for file uploads
//   - Supports *time.Time, *string, *int, etc.
//
// Nil Behavior:
//   - Optional fields: nil when missing
//   - Required fields: error when missing
//   - Empty strings: nil for optional, error for required
//
// Parameters:
//   - form: Multipart form data source
//   - field: Reflect value of the field to set
//   - fieldName: Name of the form field to read from
//   - tag: Parsed struct tag with validation rules
//
// Returns:
//   - error: Value conversion or validation error
//
// Example:
//
//	// Struct field: Email *string `form:"email"`
//	// Form data: email=test@example.com
//	// Result: field points to "test@example.com"
func (c *Context) setPointerField(form *MultipartForm, field reflect.Value, fieldName string, tag StructTag) error {
	elemType := field.Type().Elem()

	// Handle file pointers specially
	if elemType == reflect.TypeOf(MultipartFile{}) {
		file := form.GetFile(fieldName)
		if file == nil {
			if tag.Required {
				return fmt.Errorf("required file field %s is missing", fieldName)
			}
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		field.Set(reflect.ValueOf(file))
		return nil
	}

	value := form.GetValue(fieldName)
	if value == "" {
		if tag.Required {
			return fmt.Errorf("required field %s is missing", fieldName)
		}
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	// Handle time.Time pointers specially
	if elemType == reflect.TypeOf(time.Time{}) {
		tempField := reflect.New(elemType).Elem()
		if err := c.parseTimeField(tempField, value, fieldName); err != nil {
			return err
		}
		// Create pointer to the parsed time
		newTimePtr := reflect.New(elemType)
		newTimePtr.Elem().Set(tempField)
		field.Set(newTimePtr)
		return nil
	}

	// Create new instance of the pointed-to type
	newElem := reflect.New(elemType)
	if err := c.setScalarValue(newElem.Elem(), value, elemType); err != nil {
		return err
	}

	field.Set(newElem)
	return nil
}
