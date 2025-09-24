# file-handling.md

## File Handling in Blaze

Blaze provides comprehensive file handling capabilities including file uploads, downloads, serving static files, streaming, and multipart form processing. This documentation covers all aspects of working with files in your Blaze applications.

### Overview

The Blaze framework offers several approaches to file handling:

- **File Uploads**: Handle multipart form file uploads with validation and storage
- **File Downloads**: Serve files for download with proper headers
- **Static File Serving**: Serve static assets like images, CSS, and JavaScript
- **File Streaming**: Stream large files with support for HTTP range requests
- **Multipart Form Processing**: Parse and validate complex multipart forms

### Basic File Operations

#### Serving Files

The simplest way to serve a file is using the `SendFile` or `ServeFile` methods :

```go
app.GET("/download/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./uploads/%s", filename)
    
    // Check if file exists
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    return c.SendFile(filepath)
})
```

#### File Information

Get file information before serving :

```go
app.GET("/file-info/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./files/%s", filename)
    
    info, err := c.GetFileInfo(filepath)
    if err != nil {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    return c.JSON(blaze.Map{
        "name":     info.Name(),
        "size":     info.Size(),
        "modified": info.ModTime(),
        "is_dir":   info.IsDir(),
    })
})
```

### File Downloads

#### Standard Download

Force file download with custom filename :

```go
app.GET("/download/report", func(c *blaze.Context) error {
    filepath := "./reports/monthly_report.pdf"
    filename := "Monthly_Report_2024.pdf"
    
    return c.ServeFileDownload(filepath, filename)
})
```

#### Inline File Display

Serve files for inline display (like images in browser) :

```go
app.GET("/view/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./images/%s", filename)
    
    return c.ServeFileInline(filepath)
})
```

### File Streaming

For large files, use streaming to support HTTP range requests :

```go
app.GET("/stream/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./videos/%s", filename)
    
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    return c.StreamFile(filepath)
})
```

The streaming functionality automatically handles:
- HTTP range requests for partial content
- Proper MIME type detection
- Content-Length headers
- Accept-Ranges headers

### File Uploads

#### Basic File Upload

Handle single file uploads :

```go
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    // Save to uploads directory
    savePath, err := c.SaveUploadedFileToDir(file, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save file"})
    }
    
    return c.JSON(blaze.Map{
        "message":  "File uploaded successfully",
        "filename": file.Filename,
        "size":     file.Size,
        "path":     savePath,
    })
})
```

#### Multiple File Uploads

Handle multiple files from the same field :

```go
app.POST("/upload-multiple", func(c *blaze.Context) error {
    files, err := c.FormFiles("files")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No files uploaded"})
    }
    
    var uploadedFiles []blaze.Map
    
    for _, file := range files {
        savePath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            continue // Skip failed uploads
        }
        
        uploadedFiles = append(uploadedFiles, blaze.Map{
            "filename": file.Filename,
            "size":     file.Size,
            "path":     savePath,
            "type":     file.ContentType,
        })
    }
    
    return c.JSON(blaze.Map{
        "message": "Files uploaded successfully",
        "files":   uploadedFiles,
        "count":   len(uploadedFiles),
    })
})
```

### Multipart Form Configuration

#### Default Configuration

Use default multipart settings :

```go
app.POST("/form-upload", func(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid form data"})
    }
    
    // Clean up temporary files when done
    defer form.Cleanup()
    
    return c.JSON(blaze.Map{
        "files_count": form.GetFileCount(),
        "total_size":  form.GetTotalSize(),
        "fields":      len(form.Value),
    })
})
```

#### Custom Configuration

Configure multipart parsing with custom settings :

```go
app.POST("/custom-upload", func(c *blaze.Context) error {
    config := &blaze.MultipartConfig{
        MaxMemory:   10 << 20, // 10MB
        MaxFileSize: 50 << 20, // 50MB
        MaxFiles:    5,
        AllowedExtensions: []string{".jpg", ".png", ".pdf"},
        AllowedMimeTypes: []string{
            "image/jpeg", "image/png", "application/pdf",
        },
        KeepInMemory: false,
        AutoCleanup:  true,
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    defer form.Cleanup()
    
    // Process form
    return c.JSON(blaze.Map{"message": "Form processed successfully"})
})
```

#### Production Configuration

Use production-ready multipart settings :

```go
func setupMultipartRoutes(app *blaze.App) {
    // Use production config
    config := blaze.ProductionMultipartConfig()
    
    app.POST("/production-upload", func(c *blaze.Context) error {
        form, err := c.MultipartFormWithConfig(config)
        if err != nil {
            return c.Status(400).JSON(blaze.Map{"error": err.Error()})
        }
        defer form.Cleanup()
        
        // Process files safely
        for fieldName, files := range form.File {
            for _, file := range files {
                // Validate file type
                if !file.IsImage() && !file.IsDocument() {
                    continue
                }
                
                // Save with unique filename
                savePath, err := file.SaveWithUniqueFilename("./secure_uploads")
                if err != nil {
                    continue
                }
                
                log.Printf("Saved file: %s", savePath)
            }
        }
        
        return c.JSON(blaze.Map{"message": "Files processed successfully"})
    })
}
```

### File Validation Middleware

#### File Size Limits

Restrict upload file sizes :

```go
// Limit files to 10MB
app.Use(blaze.FileSizeLimitMiddleware(10 << 20))

app.POST("/upload", func(c *blaze.Context) error {
    // File size already validated by middleware
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    return c.JSON(blaze.Map{"message": "File within size limits"})
})
```

#### File Type Restrictions

Restrict file types using middleware :

```go
// Allow only images
app.Use(blaze.ImageOnlyMiddleware())

// Or allow only documents
app.Use(blaze.DocumentOnlyMiddleware())

// Or create custom restrictions
app.Use(blaze.FileTypeMiddleware(
    []string{".jpg", ".png", ".gif"}, // allowed extensions
    []string{"image/jpeg", "image/png", "image/gif"}, // allowed MIME types
))
```

#### Custom File Validation

Create custom validation middleware :

```go
func CustomFileValidationMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            if !c.IsMultipartForm() {
                return next(c)
            }
            
            form, err := c.MultipartForm()
            if err != nil {
                return c.Status(400).JSON(blaze.Map{"error": "Invalid form"})
            }
            defer form.Cleanup()
            
            // Custom validation logic
            for _, files := range form.File {
                for _, file := range files {
                    // Check file name
                    if strings.Contains(file.Filename, "../") {
                        return c.Status(400).JSON(blaze.Map{
                            "error": "Invalid filename",
                        })
                    }
                    
                    // Check file content (simplified example)
                    if file.Size == 0 {
                        return c.Status(400).JSON(blaze.Map{
                            "error": "Empty file not allowed",
                        })
                    }
                }
            }
            
            return next(c)
        }
    }
}
```

### Advanced File Operations

#### Working with Form Data

Process mixed form data with files and text fields :

```go
app.POST("/profile-update", func(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid form data"})
    }
    defer form.Cleanup()
    
    // Get text fields
    username := form.GetValue("username")
    email := form.GetValue("email")
    
    // Get uploaded avatar
    avatar := form.GetFile("avatar")
    
    var avatarPath string
    if avatar != nil {
        avatarPath, err = avatar.SaveWithUniqueFilename("./avatars")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{"error": "Failed to save avatar"})
        }
    }
    
    return c.JSON(blaze.Map{
        "username":    username,
        "email":       email,
        "avatar_path": avatarPath,
        "message":     "Profile updated successfully",
    })
})
```

#### File Processing Pipeline

Create a comprehensive file processing handler :

```go
func FileProcessingHandler(uploadDir string) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        config := &blaze.MultipartConfig{
            MaxMemory:   32 << 20,
            MaxFileSize: 100 << 20,
            MaxFiles:    10,
            AllowedExtensions: []string{
                ".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".docx",
            },
            KeepInMemory: false,
            AutoCleanup:  true,
        }
        
        form, err := c.MultipartFormWithConfig(config)
        if err != nil {
            return c.Status(400).JSON(blaze.Map{"error": err.Error()})
        }
        defer form.Cleanup()
        
        var processedFiles []blaze.Map
        var errors []string
        
        for fieldName, files := range form.File {
            for _, file := range files {
                result := blaze.Map{
                    "field":    fieldName,
                    "filename": file.Filename,
                    "size":     file.Size,
                    "type":     file.ContentType,
                }
                
                // Validate file
                if file.Size == 0 {
                    errors = append(errors, fmt.Sprintf("Empty file: %s", file.Filename))
                    continue
                }
                
                // Save file
                savePath, err := file.SaveWithUniqueFilename(uploadDir)
                if err != nil {
                    errors = append(errors, fmt.Sprintf("Failed to save %s: %v", file.Filename, err))
                    continue
                }
                
                result["saved_path"] = savePath
                result["status"] = "success"
                
                // Additional processing based on file type
                if file.IsImage() {
                    result["category"] = "image"
                    // Add image-specific processing here
                } else if file.IsDocument() {
                    result["category"] = "document"
                    // Add document-specific processing here
                }
                
                processedFiles = append(processedFiles, result)
            }
        }
        
        return c.JSON(blaze.Map{
            "files":         processedFiles,
            "processed":     len(processedFiles),
            "total_size":    form.GetTotalSize(),
            "errors":        errors,
            "has_errors":    len(errors) > 0,
        })
    }
}
```

### File Upload with Progress Tracking

For large file uploads, implement progress tracking :

```go
app.POST("/upload-with-progress", func(c *blaze.Context) error {
    // Set longer timeout for large uploads
    ctx, cancel := c.WithTimeout(300 * time.Second)
    defer cancel()
    
    // Check if client is still connected
    select {
    case <-ctx.Done():
        return c.Status(408).JSON(blaze.Map{"error": "Upload timeout"})
    default:
    }
    
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Upload failed"})
    }
    defer form.Cleanup()
    
    file := form.GetFile("file")
    if file == nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file provided"})
    }
    
    // Save with progress logging
    savePath, err := file.SaveWithUniqueFilename("./large_uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save file"})
    }
    
    return c.JSON(blaze.Map{
        "message":     "Large file uploaded successfully",
        "filename":    file.Filename,
        "size":        file.Size,
        "saved_path":  savePath,
        "upload_time": time.Now().Format(time.RFC3339),
    })
})
```

### Static File Serving

While not explicitly shown in the context methods, you can create static file handlers :

```go
func StaticFileHandler(directory string) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := fmt.Sprintf("%s/%s", directory, filename)
        
        // Security check - prevent directory traversal
        if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
            return c.Status(403).JSON(blaze.Map{"error": "Access denied"})
        }
        
        if !c.FileExists(filepath) {
            return c.Status(404).JSON(blaze.Map{"error": "File not found"})
        }
        
        return c.ServeFileInline(filepath)
    }
}

// Usage
app.GET("/static/:filename", StaticFileHandler("./public"))
app.GET("/images/:filename", StaticFileHandler("./uploads/images"))
```

### Best Practices

#### Security Considerations

1. **File Type Validation**: Always validate file types on both client and server
2. **File Size Limits**: Implement appropriate size limits to prevent DoS attacks
3. **Filename Sanitization**: Never trust user-provided filenames
4. **Directory Traversal Protection**: Prevent access to files outside designated directories
5. **Virus Scanning**: Consider integrating virus scanning for uploaded files

#### Performance Optimization

1. **Use Streaming**: For large files, use streaming to reduce memory usage
2. **Implement Caching**: Cache frequently accessed static files
3. **Compression**: Use gzip compression for text-based files
4. **CDN Integration**: Serve static files from a CDN for better performance

#### Error Handling

Always implement comprehensive error handling for file operations :

```go
app.POST("/robust-upload", func(c *blaze.Context) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Upload panic recovered: %v", r)
            c.Status(500).JSON(blaze.Map{"error": "Upload failed unexpectedly"})
        }
    }()
    
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No file uploaded",
            "code":  "NO_FILE",
        })
    }
    
    // Validate file
    if file.Size > 10<<20 { // 10MB limit
        return c.Status(413).JSON(blaze.Map{
            "error": "File too large",
            "code":  "FILE_TOO_LARGE",
            "limit": "10MB",
        })
    }
    
    // Save file with error handling
    savePath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
    if err != nil {
        log.Printf("Failed to save file %s: %v", file.Filename, err)
        return c.Status(500).JSON(blaze.Map{
            "error": "Failed to save file",
            "code":  "SAVE_FAILED",
        })
    }
    
    return c.JSON(blaze.Map{
        "message":   "File uploaded successfully",
        "filename":  file.Filename,
        "save_path": savePath,
        "size":      file.Size,
    })
})
```

This comprehensive guide covers all aspects of file handling in Blaze, from basic operations to advanced use cases with proper error handling and security considerations.
