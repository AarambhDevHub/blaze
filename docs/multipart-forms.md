# Multipart Forms

The Blaze web framework provides comprehensive support for handling multipart form data and file uploads. This guide covers everything you need to know about working with multipart forms, including configuration, file handling, validation, and advanced features.

## Table of Contents

1. [Overview](#overview)
2. [Basic Usage](#basic-usage)
3. [Configuration](#configuration)
4. [File Upload Handling](#file-upload-handling)
5. [Form Data Binding](#form-data-binding)
6. [Validation](#validation)
7. [Middleware](#middleware)
8. [Advanced Features](#advanced-features)
9. [Best Practices](#best-practices)
10. [Examples](#examples)

## Overview

Multipart forms are commonly used for file uploads and mixed data submissions in web applications. Blaze provides robust multipart form handling with features like automatic file validation, memory management, temporary file handling, and struct binding.

Key features:
- Automatic multipart form parsing
- File size and type validation
- Memory-efficient handling of large files
- Automatic cleanup of temporary files
- Struct binding with validation tags
- Comprehensive middleware support

## Basic Usage

### Checking for Multipart Forms

```go
func uploadHandler(c *blaze.Context) error {
    if !c.IsMultipartForm() {
        return c.Status(400).JSON(blaze.Map{
            "error": "Expected multipart form data",
        })
    }
    
    // Handle multipart form
    return c.Text("Multipart form detected")
}
```

### Parsing Multipart Forms

```go
func uploadHandler(c *blaze.Context) error {
    // Parse multipart form with default configuration
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Failed to parse multipart form",
        })
    }
    
    // Access form values
    username := form.GetValue("username")
    
    return c.JSON(blaze.Map{
        "username": username,
        "files": form.GetFileCount(),
    })
}
```

## Configuration

### MultipartConfig Structure

The `MultipartConfig` struct provides comprehensive configuration options for multipart form handling :

```go
type MultipartConfig struct {
    MaxMemory        int64    // Maximum memory for parsing (bytes)
    MaxFileSize      int64    // Maximum file size allowed (bytes)
    MaxFiles         int      // Maximum number of files
    TempDir          string   // Temporary directory for large files
    AllowedExtensions []string // Allowed file extensions
    AllowedMimeTypes []string  // Allowed MIME types
    KeepInMemory     bool     // Keep files in memory vs disk
    AutoCleanup      bool     // Auto cleanup temp files
}
```

### Default Configuration

```go
func DefaultMultipartConfig() *blaze.MultipartConfig {
    return &blaze.MultipartConfig{
        MaxMemory:         32 << 20,  // 32 MB
        MaxFileSize:       100 << 20, // 100 MB
        MaxFiles:          10,
        TempDir:           os.TempDir(),
        AllowedExtensions: []string{}, // Allow all
        AllowedMimeTypes:  []string{}, // Allow all
        KeepInMemory:      true,
        AutoCleanup:       true,
    }
}
```

### Production Configuration

```go
func ProductionMultipartConfig() *blaze.MultipartConfig {
    return &blaze.MultipartConfig{
        MaxMemory:         10 << 20,  // 10 MB
        MaxFileSize:       50 << 20,  // 50 MB
        MaxFiles:          5,
        TempDir:           "/tmp/uploads",
        AllowedExtensions: []string{
            ".jpg", ".jpeg", ".png", ".gif", ".pdf", 
            ".txt", ".csv", ".doc", ".docx",
        },
        AllowedMimeTypes: []string{
            "image/jpeg", "image/png", "image/gif",
            "application/pdf", "text/plain",
        },
        KeepInMemory: false,
        AutoCleanup:  true,
    }
}
```

### Using Custom Configuration

```go
func uploadHandler(c *blaze.Context) error {
    config := &blaze.MultipartConfig{
        MaxMemory:   5 << 20,  // 5 MB
        MaxFileSize: 10 << 20, // 10 MB
        MaxFiles:    3,
        AllowedExtensions: []string{".jpg", ".png", ".pdf"},
        KeepInMemory: false,
        AutoCleanup:  true,
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    // Process form
    return c.JSON(blaze.Map{"status": "success"})
}
```

## File Upload Handling

### Single File Upload

```go
func singleFileUpload(c *blaze.Context) error {
    // Get single file
    file, err := c.FormFile("avatar")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No file uploaded",
        })
    }
    
    // Validate file
    if file.Size > 5<<20 { // 5MB
        return c.Status(400).JSON(blaze.Map{
            "error": "File too large",
        })
    }
    
    // Save file
    savedPath, err := c.SaveUploadedFileToDir(file, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{
            "error": "Failed to save file",
        })
    }
    
    return c.JSON(blaze.Map{
        "filename": file.Filename,
        "size":     file.Size,
        "path":     savedPath,
        "type":     file.ContentType,
    })
}
```

### Multiple File Upload

```go
func multipleFileUpload(c *blaze.Context) error {
    // Get all files for field name
    files, err := c.FormFiles("documents")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No files uploaded",
        })
    }
    
    var savedFiles []blaze.Map
    
    for _, file := range files {
        // Validate each file
        if !file.IsDocument() {
            continue // Skip non-documents
        }
        
        // Save with unique filename
        savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            continue // Skip failed uploads
        }
        
        savedFiles = append(savedFiles, blaze.Map{
            "filename":     file.Filename,
            "size":        file.Size,
            "path":        savedPath,
            "contentType": file.ContentType,
        })
    }
    
    return c.JSON(blaze.Map{
        "uploaded": len(savedFiles),
        "files":    savedFiles,
    })
}
```

### File Operations

```go
func fileOperations(c *blaze.Context) error {
    file, err := c.FormFile("upload")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    // File information
    info := blaze.Map{
        "filename":    file.Filename,
        "size":        file.Size,
        "contentType": file.ContentType,
        "extension":   file.GetExtension(),
        "mimeType":    file.GetMimeType(),
        "isImage":     file.IsImage(),
        "isDocument":  file.IsDocument(),
    }
    
    // Different save options
    switch c.Query("saveType") {
    case "exact":
        // Save to exact path
        err = c.SaveUploadedFile(file, "./uploads/exact.jpg")
    case "directory":
        // Save to directory with original filename
        _, err = c.SaveUploadedFileToDir(file, "./uploads")
    case "unique":
        // Save with unique filename
        _, err = c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
    }
    
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(info)
}
```

## Form Data Binding

### Basic Struct Binding

Blaze supports automatic binding of multipart form data to structs using reflection and struct tags :

```go
type UserProfile struct {
    Username string                `form:"username,required"`
    Email    string                `form:"email,required"`
    Age      int                   `form:"age"`
    Bio      string                `form:"bio,maxsize=500"`
    Avatar   *blaze.MultipartFile  `form:"avatar"`
}

func updateProfile(c *blaze.Context) error {
    var profile UserProfile
    
    if err := c.BindMultipartForm(&profile); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid form data",
            "details": err.Error(),
        })
    }
    
    // Process the bound data
    if profile.Avatar != nil {
        avatarPath, _ := c.SaveUploadedFileToDir(profile.Avatar, "./avatars")
        // Save avatar path to database
    }
    
    return c.JSON(blaze.Map{
        "username": profile.Username,
        "email":    profile.Email,
        "age":      profile.Age,
    })
}
```

### Advanced Struct Tags

Blaze supports comprehensive struct tags for validation and binding :

```go
type ProductForm struct {
    Name        string                   `form:"name,required,maxsize=100"`
    Description string                   `form:"description,maxsize=1000"`
    Price       float64                  `form:"price,required"`
    Category    string                   `form:"category,required"`
    Tags        []string                 `form:"tags"`
    Images      []*blaze.MultipartFile   `form:"images"`
    IsActive    bool                     `form:"is_active,default=true"`
    LaunchDate  *time.Time               `form:"launch_date"`
}

func createProduct(c *blaze.Context) error {
    var product ProductForm
    
    if err := c.BindMultipartForm(&product); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    // Process images
    var imagePaths []string
    for _, image := range product.Images {
        if image.IsImage() {
            path, _ := c.SaveUploadedFileWithUniqueFilename(image, "./products")
            imagePaths = append(imagePaths, path)
        }
    }
    
    return c.JSON(blaze.Map{
        "product":    product.Name,
        "images":     len(imagePaths),
        "launchDate": product.LaunchDate,
    })
}
```

### Supported Struct Tag Options

- `required`: Field is mandatory
- `maxsize=N`: Maximum size for strings/files
- `minsize=N`: Minimum size for strings/files
- `default=value`: Default value if empty
- Field name mapping: `form:"custom_name"`
- Skip field: `form:"-"`

### Time Field Handling

Blaze supports comprehensive time parsing for form fields :

```go
type EventForm struct {
    Name      string    `form:"name,required"`
    StartTime time.Time `form:"start_time,required"`
    EndTime   *time.Time `form:"end_time"`
}

// Supported time formats:
// - HTML datetime-local: "2006-01-02T15:04"
// - ISO 8601: "2006-01-02T15:04:05Z07:00"
// - SQL datetime: "2006-01-02 15:04:05"
// - Date only: "2006-01-02"
// - Unix timestamp: "1609459200"
// - And many more...
```

## Validation

### Built-in Validation

Blaze provides built-in validation for multipart forms:

```go
func validateUpload(c *blaze.Context) error {
    config := &blaze.MultipartConfig{
        MaxFileSize: 5 << 20, // 5MB
        MaxFiles:    3,
        AllowedExtensions: []string{".jpg", ".png", ".pdf"},
        AllowedMimeTypes: []string{
            "image/jpeg", "image/png", "application/pdf",
        },
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        // Validation errors are automatically returned
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(blaze.Map{"status": "valid"})
}
```

### Custom Validation

```go
func customValidation(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Custom validation logic
    for fieldName, files := range form.File {
        for _, file := range files {
            // Custom file validation
            if file.Size > 10<<20 { // 10MB
                return c.Status(400).JSON(blaze.Map{
                    "error": fmt.Sprintf("File %s too large", file.Filename),
                })
            }
            
            // Custom content validation
            if strings.Contains(strings.ToLower(file.Filename), "unsafe") {
                return c.Status(400).JSON(blaze.Map{
                    "error": "Unsafe filename detected",
                })
            }
        }
    }
    
    return c.JSON(blaze.Map{"status": "validated"})
}
```

## Middleware

### MultipartMiddleware

The `MultipartMiddleware` pre-processes multipart forms for reuse :

```go
func setupRoutes(app *blaze.App) {
    // Apply to all routes
    app.Use(blaze.MultipartMiddleware(blaze.DefaultMultipartConfig()))
    
    // Apply to specific route group
    uploadGroup := app.Group("/upload")
    uploadGroup.Use(blaze.MultipartMiddleware(&blaze.MultipartConfig{
        MaxFileSize: 10 << 20,
        MaxFiles:    5,
        AllowedExtensions: []string{".jpg", ".png"},
        AutoCleanup: true,
    }))
    
    uploadGroup.POST("/avatar", uploadAvatar)
    uploadGroup.POST("/documents", uploadDocuments)
}
```

### File Type Middleware

```go
// Restrict to images only
app.Use(blaze.ImageOnlyMiddleware())

// Restrict to documents only
app.Use(blaze.DocumentOnlyMiddleware())

// Custom file type restriction
app.Use(blaze.FileTypeMiddleware(
    []string{".jpg", ".png", ".pdf"},           // Extensions
    []string{"image/jpeg", "image/png", "application/pdf"}, // MIME types
))
```

### File Size Middleware

```go
// Limit total request size
app.Use(blaze.FileSizeLimitMiddleware(50 << 20)) // 50MB limit
```

### Logging Middleware

```go
// Log multipart form details
app.Use(blaze.MultipartLoggingMiddleware())
```

## Advanced Features

### Memory Management

Blaze provides intelligent memory management for multipart forms:

```go
func memoryEfficientUpload(c *blaze.Context) error {
    config := &blaze.MultipartConfig{
        MaxMemory:    1 << 20,  // 1MB in memory
        KeepInMemory: false,    // Use temp files for large files
        AutoCleanup:  true,     // Auto cleanup temp files
        TempDir:      "/tmp/uploads",
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Form automatically manages memory vs disk storage
    // Cleanup is handled automatically
    
    return c.JSON(blaze.Map{
        "totalSize": form.GetTotalSize(),
        "fileCount": form.GetFileCount(),
    })
}
```

### Manual Cleanup

```go
func manualCleanup(c *blaze.Context) error {
    config := &blaze.MultipartConfig{
        AutoCleanup: false, // Disable auto cleanup
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Process files
    processed := processFiles(form)
    
    // Manual cleanup when done
    if err := form.Cleanup(); err != nil {
        log.Printf("Cleanup error: %v", err)
    }
    
    return c.JSON(blaze.Map{"processed": processed})
}
```

### Form Value Access

```go
func formValueAccess(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Different ways to access form values
    username := form.GetValue("username")         // First value
    tags := form.GetValues("tags")                // All values
    hasEmail := form.HasValue("email")            // Check existence
    
    // File access
    avatar := form.GetFile("avatar")              // First file
    documents := form.GetFiles("documents")       // All files
    hasAvatar := form.HasFile("avatar")           // Check existence
    
    return c.JSON(blaze.Map{
        "username":  username,
        "tags":      tags,
        "hasEmail":  hasEmail,
        "hasAvatar": hasAvatar,
        "docCount":  len(documents),
    })
}
```

## Best Practices

### 1. Always Use Configuration

```go
// Good: Use appropriate configuration
config := blaze.ProductionMultipartConfig()
form, err := c.MultipartFormWithConfig(config)

// Avoid: Using defaults in production without consideration
form, err := c.MultipartForm() // May not have appropriate limits
```

### 2. Validate Early

```go
func bestPracticeHandler(c *blaze.Context) error {
    // Validate content type first
    if !c.IsMultipartForm() {
        return c.Status(400).JSON(blaze.Map{
            "error": "Expected multipart form data",
        })
    }
    
    // Use restrictive config
    config := &blaze.MultipartConfig{
        MaxFileSize: 5 << 20,
        MaxFiles:    3,
        AllowedExtensions: []string{".jpg", ".png"},
        AutoCleanup: true,
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid form data",
            "details": err.Error(),
        })
    }
    
    // Additional custom validation
    return processValidatedForm(form)
}
```

### 3. Handle Errors Gracefully

```go
func errorHandling(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        // Provide specific error messages
        switch {
        case strings.Contains(err.Error(), "too large"):
            return c.Status(413).JSON(blaze.Map{
                "error": "File too large",
                "maxSize": "10MB",
            })
        case strings.Contains(err.Error(), "not allowed"):
            return c.Status(400).JSON(blaze.Map{
                "error": "File type not allowed",
                "allowed": []string{".jpg", ".png", ".pdf"},
            })
        default:
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid form data",
            })
        }
    }
    
    return c.JSON(blaze.Map{"status": "success"})
}
```

### 4. Use Middleware for Common Validations

```go
func setupApp() *blaze.App {
    app := blaze.New()
    
    // Global multipart middleware
    app.Use(blaze.MultipartMiddleware(blaze.DefaultMultipartConfig()))
    
    // Specific restrictions for different endpoints
    imageUpload := app.Group("/images")
    imageUpload.Use(blaze.ImageOnlyMiddleware())
    imageUpload.Use(blaze.FileSizeLimitMiddleware(5 << 20))
    
    docUpload := app.Group("/documents")
    docUpload.Use(blaze.DocumentOnlyMiddleware())
    docUpload.Use(blaze.FileSizeLimitMiddleware(50 << 20))
    
    return app
}
```

### 5. Sanitize File Names

```go
func saveFilesSafely(files []*blaze.MultipartFile, dir string) error {
    for _, file := range files {
        // Use unique filename to avoid conflicts and security issues
        safePath, err := file.SaveWithUniqueFilename(dir)
        if err != nil {
            return err
        }
        
        log.Printf("Saved file: %s", safePath)
    }
    return nil
}
```

## Examples

### Complete File Upload API

```go
func fileUploadAPI(app *blaze.App) {
    // Configure multipart middleware
    config := &blaze.MultipartConfig{
        MaxMemory:   10 << 20,  // 10MB
        MaxFileSize: 50 << 20,  // 50MB
        MaxFiles:    5,
        AllowedExtensions: []string{
            ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".csv",
        },
        KeepInMemory: false,
        AutoCleanup:  true,
        TempDir:      "/tmp/uploads",
    }
    
    upload := app.Group("/api/upload")
    upload.Use(blaze.MultipartMiddleware(config))
    upload.Use(blaze.MultipartLoggingMiddleware())
    
    // Single file upload
    upload.POST("/single", func(c *blaze.Context) error {
        file, err := c.FormFile("file")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No file provided",
            })
        }
        
        savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to save file",
            })
        }
        
        return c.JSON(blaze.Map{
            "filename":    file.Filename,
            "size":        file.Size,
            "contentType": file.ContentType,
            "path":        savedPath,
        })
    })
    
    // Multiple file upload
    upload.POST("/multiple", func(c *blaze.Context) error {
        form, err := c.MultipartForm()
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": err.Error(),
            })
        }
        
        var results []blaze.Map
        
        for fieldName, files := range form.File {
            for _, file := range files {
                savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
                if err != nil {
                    continue
                }
                
                results = append(results, blaze.Map{
                    "field":       fieldName,
                    "filename":    file.Filename,
                    "size":        file.Size,
                    "contentType": file.ContentType,
                    "path":        savedPath,
                })
            }
        }
        
        return c.JSON(blaze.Map{
            "uploaded": len(results),
            "files":    results,
            "formData": form.Value,
        })
    })
    
    // Form with file binding
    upload.POST("/profile", func(c *blaze.Context) error {
        type ProfileForm struct {
            Name     string               `form:"name,required,maxsize=100"`
            Email    string               `form:"email,required"`
            Bio      string               `form:"bio,maxsize=500"`
            Avatar   *blaze.MultipartFile `form:"avatar"`
            Age      int                  `form:"age"`
            IsActive bool                 `form:"active,default=true"`
        }
        
        var profile ProfileForm
        if err := c.BindMultipartForm(&profile); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid form data",
                "details": err.Error(),
            })
        }
        
        var avatarPath string
        if profile.Avatar != nil && profile.Avatar.IsImage() {
            path, err := c.SaveUploadedFileToDir(profile.Avatar, "./avatars")
            if err == nil {
                avatarPath = path
            }
        }
        
        return c.JSON(blaze.Map{
            "profile": blaze.Map{
                "name":     profile.Name,
                "email":    profile.Email,
                "bio":      profile.Bio,
                "age":      profile.Age,
                "active":   profile.IsActive,
                "avatar":   avatarPath,
            },
        })
    })
}
```

This comprehensive documentation covers all aspects of multipart form handling in the Blaze web framework, from basic usage to advanced features and best practices.