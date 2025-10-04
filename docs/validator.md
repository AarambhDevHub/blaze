# Validation

Blaze provides comprehensive request validation using the powerful `go-playground/validator/v10` library. The validation system is deeply integrated into the framework, allowing for automatic validation of request bodies, form data, and multipart forms with minimal code.

## Table of Contents

- [Overview](#overview)
- [Validator Structure](#validator-structure)
- [Validation Tags](#validation-tags)
- [Context Validation Methods](#context-validation-methods)
- [Validation Middleware](#validation-middleware)
- [Error Handling](#error-handling)
- [Custom Validators](#custom-validators)
- [Common Validation Patterns](#common-validation-patterns)
- [Production Best Practices](#production-best-practices)

## Overview

Blaze's validation system provides:

- **Struct Validation**: Validate entire request structs using tags
- **Single Variable Validation**: Validate individual values
- **Automatic Error Formatting**: User-friendly error messages
- **Custom Validators**: Register custom validation rules
- **Integrated Binding**: Bind and validate in one call
- **JSON Field Names**: Error messages use JSON field names

## Validator Structure

### Validator Type

```go
type Validator struct {
    validate *validator.Validate
}
```

The `Validator` wraps go-playground's validator with Blaze-specific enhancements:
- Automatic JSON field name resolution in errors
- User-friendly error message formatting
- Integration with context methods

### ValidationError

Represents a single field validation error:

```go
type ValidationError struct {
    Field   string      `json:"field"`
    Tag     string      `json:"tag"`
    Value   interface{} `json:"value,omitempty"`
    Message string      `json:"message"`
}
```

### ValidationErrors

Represents multiple validation errors:

```go
type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}

func (v *ValidationErrors) Error() string {
    var messages []string
    for _, err := range v.Errors {
        messages = append(messages, err.Message)
    }
    return strings.Join(messages, "; ")
}
```

## Validation Tags

Blaze supports all standard go-playground/validator tags:

### Common Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must be present | `validate:"required"` |
| `email` | Valid email address | `validate:"email"` |
| `min` | Minimum value/length | `validate:"min=8"` |
| `max` | Maximum value/length | `validate:"max=100"` |
| `len` | Exact length | `validate:"len=10"` |
| `eq` | Equal to value | `validate:"eq=5"` |
| `ne` | Not equal to value | `validate:"ne=0"` |
| `gt` | Greater than | `validate:"gt=0"` |
| `gte` | Greater than or equal | `validate:"gte=18"` |
| `lt` | Less than | `validate:"lt=100"` |
| `lte` | Less than or equal | `validate:"lte=150"` |
| `alpha` | Alphabetic characters only | `validate:"alpha"` |
| `alphanum` | Alphanumeric characters only | `validate:"alphanum"` |
| `numeric` | Numeric characters only | `validate:"numeric"` |
| `url` | Valid URL | `validate:"url"` |
| `uri` | Valid URI | `validate:"uri"` |
| `uuid` | Valid UUID | `validate:"uuid"` |
| `oneof` | One of specified values | `validate:"oneof=red blue green"` |

### Field Comparison Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `eqfield` | Equal to another field | `validate:"eqfield=Password"` |
| `nefield` | Not equal to another field | `validate:"nefield=OldPassword"` |
| `gtfield` | Greater than another field | `validate:"gtfield=StartDate"` |
| `gtefield` | Greater than or equal to field | `validate:"gtefield=MinValue"` |
| `ltfield` | Less than another field | `validate:"ltfield=EndDate"` |
| `ltefield` | Less than or equal to field | `validate:"ltefield=MaxValue"` |

### String Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `contains` | Contains substring | `validate:"contains=@"` |
| `containsany` | Contains any character | `validate:"containsany=abc"` |
| `excludes` | Doesn't contain substring | `validate:"excludes=admin"` |
| `startswith` | Starts with prefix | `validate:"startswith=https://"` |
| `endswith` | Ends with suffix | `validate:"endswith=.com"` |

### Date/Time Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `datetime` | Valid datetime format | `validate:"datetime=2006-01-02"` |

## Context Validation Methods

Blaze provides convenient context methods for validation:

### Bind and Validate

Automatically bind and validate in one call:

```go
type UserRegistration struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=100"`
}

app.POST("/register", func(c *blaze.Context) error {
    var reg UserRegistration
    
    // Bind and validate in one call
    if err := c.BindAndValidate(&reg); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    return c.JSON(blaze.Created(reg))
})
```

### JSON Validation

```go
app.POST("/api/user", func(c *blaze.Context) error {
    var user User
    
    // Bind JSON and validate
    if err := c.BindJSONAndValidate(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(user)
})
```

### Form Validation

```go
app.POST("/contact", func(c *blaze.Context) error {
    var form ContactForm
    
    // Bind form data and validate
    if err := c.BindFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(form)
})
```

### Multipart Form Validation

```go
type FileUploadForm struct {
    Title       string                `form:"title,required,minsize:2"`
    Description string                `form:"description,maxsize:500"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
}

app.POST("/upload", func(c *blaze.Context) error {
    var form FileUploadForm
    
    // Bind multipart form and validate
    if err := c.BindMultipartFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(form)
})
```

### Struct Validation Only

Validate a struct without binding:

```go
app.POST("/validate", func(c *blaze.Context) error {
    user := getUserFromSomewhere()
    
    // Validate existing struct
    if err := c.Validate(user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(blaze.Map{"valid": true})
})
```

### Single Variable Validation

Validate individual values:

```go
app.GET("/email/:email", func(c *blaze.Context) error {
    email := c.Param("email")
    
    // Validate single variable
    if err := c.ValidateVar(email, "email"); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid email address",
        })
    }
    
    return c.JSON(blaze.Map{"email": email})
})
```

**Available Context Methods:**
- `BindAndValidate(v interface{}) error` - Auto-detect, bind and validate
- `BindJSONAndValidate(v interface{}) error` - Bind JSON and validate
- `BindFormAndValidate(v interface{}) error` - Bind form and validate
- `BindMultipartFormAndValidate(v interface{}) error` - Bind multipart and validate
- `Validate(v interface{}) error` - Validate struct without binding
- `ValidateVar(field interface{}, tag string) error` - Validate single variable

## Validation Middleware

### Global Validation Middleware

Enable automatic validation error handling:

```go
app := blaze.New()

// Add validation middleware
app.Use(blaze.ValidationMiddleware())

// Now all validation errors are automatically handled
app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    // Validation errors automatically return 400 with details
    if err := c.BindAndValidate(&user); err != nil {
        return err // Middleware handles this
    }
    
    return c.JSON(user)
})
```

### Route-Specific Validation

Apply validation to specific routes:

```go
// Custom validation function
validateUser := func(c *blaze.Context) error {
    var user User
    if err := c.BindJSON(&user); err != nil {
        return err
    }
    
    if user.Age < 18 {
        return blaze.ErrValidation("User must be 18 or older")
    }
    
    return nil
}

// Apply to route
app.POST("/users", createUserHandler,
    blaze.WithMiddleware(blaze.ValidateRequest(validateUser)))
```

## Error Handling

### Validation Error Response

Validation errors return structured JSON:

```go
{
    "success": false,
    "error": "Validation failed",
    "details": [
        {
            "field": "email",
            "tag": "email",
            "message": "email must be a valid email address"
        },
        {
            "field": "password",
            "tag": "min",
            "message": "password must be at least 8 characters long"
        }
    ]
}
```

### Custom Error Handling

Handle validation errors with custom logic:

```go
app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    if err := c.BindJSONAndValidate(&user); err != nil {
        // Check if it's a validation error
        if validationErr, ok := err.(*blaze.ValidationErrors); ok {
            return c.Status(422).JSON(blaze.Map{
                "error": "Validation failed",
                "fields": validationErr.Errors,
            })
        }
        
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid request",
        })
    }
    
    return c.JSON(user)
})
```

### Error Message Formatting

Blaze automatically formats validation error messages:

```go
// Tag: required
"name is required"

// Tag: email
"email must be a valid email address"

// Tag: min=8
"password must be at least 8 characters long"

// Tag: max=100
"bio must be at most 100 characters long"

// Tag: gte=18
"age must be greater than or equal to 18"

// Tag: oneof=red blue green
"color must be one of red blue green"
```

## Custom Validators

### Register Custom Validation

```go
validator := blaze.GetValidator()

// Register custom validation function
err := validator.RegisterValidation("username", func(fl validator.FieldLevel) bool {
    username := fl.Field().String()
    
    // Username must be alphanumeric and 3-20 characters
    if len(username) < 3 || len(username) > 20 {
        return false
    }
    
    for _, char := range username {
        if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
            return false
        }
    }
    
    return true
})

if err != nil {
    log.Fatal(err)
}

// Use custom validation
type UserSignup struct {
    Username string `json:"username" validate:"required,username"`
    Email    string `json:"email" validate:"required,email"`
}
```

### Struct-Level Validation

Register validation that checks multiple fields:

```go
type DateRange struct {
    StartDate time.Time `json:"start_date" validate:"required"`
    EndDate   time.Time `json:"end_date" validate:"required"`
}

validator := blaze.GetValidator()

// Register struct-level validation
validator.RegisterStructValidation(func(sl validator.StructLevel) {
    dateRange := sl.Current().Interface().(DateRange)
    
    if dateRange.EndDate.Before(dateRange.StartDate) {
        sl.ReportError(dateRange.EndDate, "end_date", "EndDate", 
            "gtefield", "start_date")
    }
}, DateRange{})
```

### Custom Validator Instance

Create and set a custom validator:

```go
customValidator := blaze.NewValidator()

// Register custom validations
customValidator.RegisterValidation("mycustom", myValidationFunc)

// Set as global validator
blaze.SetValidator(customValidator)
```

## Common Validation Patterns

### User Registration

```go
type UserRegistration struct {
    Username        string `json:"username" validate:"required,min=3,max=20,alphanum"`
    Email           string `json:"email" validate:"required,email"`
    Password        string `json:"password" validate:"required,min=8"`
    ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
    Age             int    `json:"age" validate:"required,gte=18,lte=100"`
    Terms           bool   `json:"terms" validate:"required,eq=true"`
}

app.POST("/register", func(c *blaze.Context) error {
    var reg UserRegistration
    
    if err := c.BindJSONAndValidate(&reg); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Registration validation failed",
            "details": err.Error(),
        })
    }
    
    // Password confirmation already validated via eqfield
    return c.Status(201).JSON(blaze.Map{"message": "User registered"})
})
```

### Product Creation

```go
type Product struct {
    Name        string  `json:"name" validate:"required,min=2,max=100"`
    Description string  `json:"description" validate:"required,min=10,max=500"`
    Price       float64 `json:"price" validate:"required,gt=0"`
    Quantity    int     `json:"quantity" validate:"required,gte=0"`
    SKU         string  `json:"sku" validate:"required,len=8,alphanum"`
    Category    string  `json:"category" validate:"required,oneof=electronics clothing food"`
}

app.POST("/products", func(c *blaze.Context) error {
    var product Product
    
    if err := c.BindJSONAndValidate(&product); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Product validation failed",
            "details": err.Error(),
        })
    }
    
    return c.Status(201).JSON(product)
})
```

### Address Validation

```go
type Address struct {
    Street     string `json:"street" validate:"required,min=5,max=100"`
    City       string `json:"city" validate:"required,min=2,max=50"`
    State      string `json:"state" validate:"required,len=2,alpha"`
    ZipCode    string `json:"zip_code" validate:"required,len=5,numeric"`
    Country    string `json:"country" validate:"required,len=2,alpha"`
}

app.POST("/addresses", func(c *blaze.Context) error {
    var address Address
    
    if err := c.BindJSONAndValidate(&address); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Address validation failed",
            "details": err.Error(),
        })
    }
    
    return c.Status(201).JSON(address)
})
```

### File Upload with Validation

```go
type DocumentUpload struct {
    Title       string                `form:"title,required,minsize:5,maxsize:100"`
    Description string                `form:"description,maxsize:500"`
    Category    string                `form:"category,required"`
    Tags        []string              `form:"tags"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Published   bool                  `form:"published"`
}

app.POST("/documents/upload", func(c *blaze.Context) error {
    var doc DocumentUpload
    
    if err := c.BindMultipartFormAndValidate(&doc); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Document validation failed",
            "details": err.Error(),
        })
    }
    
    // Additional file validation
    if doc.File.Size > 10*1024*1024 {
        return c.Status(413).JSON(blaze.Map{
            "error": "File too large (max 10MB)",
        })
    }
    
    if !doc.File.IsDocument() {
        return c.Status(400).JSON(blaze.Map{
            "error": "Only document files are allowed",
        })
    }
    
    return c.JSON(blaze.Map{"message": "Document uploaded"})
})
```

### Date Range Validation

```go
type DateRangeQuery struct {
    StartDate time.Time `json:"start_date" validate:"required"`
    EndDate   time.Time `json:"end_date" validate:"required,gtfield=StartDate"`
    Limit     int       `json:"limit" validate:"omitempty,gte=1,lte=100"`
}

app.POST("/reports", func(c *blaze.Context) error {
    var query DateRangeQuery
    
    if err := c.BindJSONAndValidate(&query); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Query validation failed",
            "details": err.Error(),
        })
    }
    
    return c.JSON(blaze.Map{"message": "Report generated"})
})
```

## Production Best Practices

### 1. Always Validate User Input

```go
app.POST("/api/data", func(c *blaze.Context) error {
    var data MyStruct
    
    // Always validate before processing
    if err := c.BindAndValidate(&data); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Safe to use validated data
    return processData(data)
})
```

### 2. Use Specific Validation Tags

```go
// Good - specific validation
type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"required,gte=18,lte=100"`
    Phone string `json:"phone" validate:"required,len=10,numeric"`
}

// Avoid - too permissive
type User struct {
    Email string `json:"email" validate:"required"`
    Age   int    `json:"age" validate:"required"`
    Phone string `json:"phone" validate:"required"`
}
```

### 3. Provide Clear Error Messages

```go
app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    if err := c.BindJSONAndValidate(&user); err != nil {
        if validationErr, ok := err.(*blaze.ValidationErrors); ok {
            return c.Status(422).JSON(blaze.Map{
                "success": false,
                "error":   "Please correct the following errors:",
                "fields":  validationErr.Errors,
            })
        }
        
        return c.Status(400).JSON(blaze.Map{
            "success": false,
            "error":   "Invalid request format",
        })
    }
    
    return c.JSON(blaze.Map{"success": true})
})
```

### 4. Validate at Multiple Levels

```go
app.POST("/transfer", func(c *blaze.Context) error {
    var transfer Transfer
    
    // Basic validation
    if err := c.BindJSONAndValidate(&transfer); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    // Business logic validation
    if transfer.Amount > getAccountBalance(transfer.FromAccount) {
        return c.Status(400).JSON(blaze.Map{
            "error": "Insufficient funds",
        })
    }
    
    if transfer.FromAccount == transfer.ToAccount {
        return c.Status(400).JSON(blaze.Map{
            "error": "Cannot transfer to same account",
        })
    }
    
    return processTransfer(transfer)
})
```

### 5. Use Validation Middleware

```go
func setupApp() *blaze.App {
    app := blaze.New()
    
    // Global validation middleware
    app.Use(blaze.ValidationMiddleware())
    
    // All routes automatically handle validation errors
    app.POST("/users", createUser)
    app.PUT("/users/:id", updateUser)
    app.POST("/products", createProduct)
    
    return app
}
```

### 6. Security Considerations

```go
type UserUpdate struct {
    Email    string `json:"email" validate:"required,email"`
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Bio      string `json:"bio" validate:"omitempty,max=500"`
    
    // Don't allow users to set these directly
    Role     string `json:"-"` // Not from JSON
    IsAdmin  bool   `json:"-"` // Not from JSON
    Password string `json:"-"` // Separate endpoint for password
}

app.PUT("/users/:id", func(c *blaze.Context) error {
    var update UserUpdate
    
    if err := c.BindJSONAndValidate(&update); err != nil {
        return c.Status(422).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    // Safely update only allowed fields
    return updateUser(c.Param("id"), update)
})
```

Blaze's validation system provides a robust, production-ready solution for validating all types of user input with minimal code while maintaining security and providing clear error feedback.