# File Handling

Blaze provides comprehensive file handling capabilities including file uploads with struct binding, downloads, serving static files, streaming, and advanced multipart form processing. This documentation covers all aspects of working with files in your Blaze applications.

## Table of Contents

- [Overview](#overview)
- [File Serving](#file-serving)
- [File Downloads](#file-downloads)
- [File Streaming](#file-streaming)
- [File Uploads](#file-uploads)
- [Multipart Form Binding](#multipart-form-binding)
- [Multipart Configuration](#multipart-configuration)
- [File Validation](#file-validation)
- [Static File Serving](#static-file-serving)
- [Best Practices](#best-practices)

## Overview

The Blaze framework offers several approaches to file handling:

- **File Uploads**: Handle multipart form file uploads with validation and storage
- **Struct Binding**: Bind uploaded files directly to Go structs with automatic validation
- **File Downloads**: Serve files for download with proper headers
- **Static File Serving**: Serve static assets like images, CSS, and JavaScript
- **File Streaming**: Stream large files with support for HTTP range requests
- **Multipart Form Processing**: Parse and validate complex multipart forms with mixed data

## File Serving

### Basic File Serving

The simplest way to serve a file is using the `SendFile` or `ServeFile` methods:

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

### File Information

Get file information before serving:

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
        "mode":     info.Mode().String(),
    })
})
```

**Available Methods:**
- `SendFile(filepath string) error` - Send file to client
- `ServeFile(filepath string) error` - Serve file with proper headers
- `FileExists(filepath string) bool` - Check if file exists
- `GetFileInfo(filepath string) (os.FileInfo, error)` - Get file metadata

## File Downloads

### Standard Download

Force file download with custom filename:

```go
app.GET("/download/report", func(c *blaze.Context) error {
    filepath := "./reports/monthly_report.pdf"
    filename := "Monthly_Report_2024.pdf"
    
    return c.ServeFileDownload(filepath, filename)
})

// Alias methods
app.GET("/download/alt", func(c *blaze.Context) error {
    return c.Download("./file.pdf", "custom_name.pdf")
})

app.GET("/attachment", func(c *blaze.Context) error {
    return c.Attachment("./file.pdf", "attached.pdf")
})
```

### Inline File Display

Serve files for inline display (like images in browser):

```go
app.GET("/view/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./images/%s", filename)
    
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    return c.ServeFileInline(filepath)
})
```

**Available Methods:**
- `ServeFileDownload(filepath, filename string) error` - Force download
- `ServeFileInline(filepath string) error` - Display inline
- `Download(filepath, filename string) error` - Alias for download
- `Attachment(filepath, filename string) error` - Alias for download

## File Streaming

For large files, use streaming to support HTTP range requests:

```go
app.GET("/stream/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := fmt.Sprintf("./videos/%s", filename)
    
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    // Supports range requests for video seeking
    return c.StreamFile(filepath)
})
```

The streaming functionality automatically handles:
- HTTP range requests for partial content (206 responses)
- Proper MIME type detection based on file extension
- Content-Length and Accept-Ranges headers
- Efficient memory usage for large files

**Supported MIME Types:**
- Images: jpg, png, gif, bmp, webp, svg, ico
- Documents: pdf, doc, docx, xls, xlsx, ppt, pptx
- Text: txt, csv, html, css, js, json, xml
- Audio: mp3, wav, ogg, m4a
- Video: mp4, avi, mkv, webm, mov
- Archives: zip, rar, tar, gz, 7z

## File Uploads

### Basic Single File Upload

Handle single file uploads:

```go
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    // Validate file
    if file.Size > 10*1024*1024 {
        return c.Status(413).JSON(blaze.Map{"error": "File too large (max 10MB)"})
    }
    
    // Save to uploads directory
    savePath, err := c.SaveUploadedFileToDir(file, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save file"})
    }
    
    return c.JSON(blaze.Map{
        "message":      "File uploaded successfully",
        "filename":     file.Filename,
        "size":         file.Size,
        "path":         savePath,
        "content_type": file.ContentType,
    })
})
```

### Multiple File Uploads

Handle multiple files from the same field:

```go
app.POST("/upload-multiple", func(c *blaze.Context) error {
    files, err := c.FormFiles("files")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No files uploaded"})
    }
    
    var uploadedFiles []blaze.Map
    
    for _, file := range files {
        // Validate each file
        if file.Size > 10*1024*1024 {
            continue // Skip files larger than 10MB
        }
        
        // Save with unique filename to avoid conflicts
        savePath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            continue // Skip failed uploads
        }
        
        uploadedFiles = append(uploadedFiles, blaze.Map{
            "filename":     file.Filename,
            "size":         file.Size,
            "path":         savePath,
            "type":         file.ContentType,
            "extension":    file.GetExtension(),
            "is_image":     file.IsImage(),
            "is_document":  file.IsDocument(),
        })
    }
    
    return c.JSON(blaze.Map{
        "message": "Files uploaded successfully",
        "files":   uploadedFiles,
        "count":   len(uploadedFiles),
    })
})
```

### File Save Methods

```go
// Save to specific path
err := c.SaveUploadedFile(file, "./uploads/myfile.pdf")

// Save to directory with original filename
savedPath, err := c.SaveUploadedFileToDir(file, "./uploads")

// Save with unique filename (timestamp-based)
uniquePath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
```

### MultipartFile Methods

```go
// File information
extension := file.GetExtension()        // ".jpg"
mimeType := file.GetMimeType()          // "image/jpeg"
isImage := file.IsImage()               // true/false
isDocument := file.IsDocument()         // true/false

// File operations
err := file.Save("./path/to/file.jpg")
path, err := file.SaveToDir("./uploads")
uniquePath, err := file.SaveWithUniqueFilename("./uploads")
err := file.Cleanup()  // Remove temporary files
```

## Multipart Form Binding

### Struct-Based File Upload with Validation

Bind multipart forms directly to Go structs with automatic validation:

```go
type FileUploadForm struct {
    Title       string                `form:"title,required,minsize:2,maxsize:100"`
    Description string                `form:"description,maxsize:500"`
    Category    string                `form:"category,default:general"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
    Files       []*blaze.MultipartFile `form:"files"`
    Priority    *int                  `form:"priority"`
    Published   bool                  `form:"published"`
    PublishDate *time.Time            `form:"publish_date"`
}

app.POST("/upload/form", func(c *blaze.Context) error {
    var form FileUploadForm
    
    // Bind and validate in one call
    if err := c.BindMultipartFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Save the main file
    if form.File != nil {
        savedPath, err := c.SaveUploadedFileWithUniqueFilename(
            form.File, "./uploads")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to save file",
            })
        }
        form.File.TempFilePath = savedPath
    }
    
    // Save additional files
    var additionalPaths []string
    for _, file := range form.Files {
        path, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            continue
        }
        additionalPaths = append(additionalPaths, path)
    }
    
    return c.JSON(blaze.Map{
        "message":          "Upload successful",
        "title":            form.Title,
        "file_path":        form.File.TempFilePath,
        "file_size":        form.File.Size,
        "additional_files": additionalPaths,
        "tags":             form.Tags,
        "published":        form.Published,
    })
})
```

### Form Tag Options

Struct tags control validation and binding behavior:

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must be present | `form:"name,required"` |
| `minsize:N` | Minimum size in bytes | `form:"bio,minsize:10"` |
| `maxsize:N` | Maximum size in bytes | `form:"description,maxsize:1000"` |
| `default:value` | Default value if empty | `form:"category,default:general"` |

**Examples:**

```go
type UserProfile struct {
    // Text fields with validation
    Name     string `form:"name,required,minsize:2,maxsize:100"`
    Email    string `form:"email,required"`
    Bio      string `form:"bio,maxsize:500"`
    
    // Optional fields with defaults
    Country  string `form:"country,default:US"`
    Language string `form:"language,default:en"`
    
    // Numeric fields
    Age      int    `form:"age,default:18"`
    Score    *int   `form:"score"`
    
    // Boolean fields
    Active   bool   `form:"active"`
    
    // Date/time fields
    BirthDate     *time.Time `form:"birth_date"`
    RegisteredAt  time.Time  `form:"registered_at"`
    
    // File uploads
    Avatar        *blaze.MultipartFile  `form:"avatar"`
    Documents     []*blaze.MultipartFile `form:"documents"`
    
    // Arrays
    Tags          []string `form:"tags"`
    Permissions   []int    `form:"permissions"`
}
```

### Supported Field Types

- **Basic Types**: `string`, `int`, `int8`-`int64`, `uint`, `uint8`-`uint64`, `float32`, `float64`, `bool`
- **Pointers**: `*string`, `*int`, `*time.Time`, etc.
- **Slices**: `[]string`, `[]int`, `[]*MultipartFile`, etc.
- **Time**: `time.Time`, `*time.Time` with multiple format support
- **Files**: `*MultipartFile`, `[]*MultipartFile`

### Time Format Support

The binding automatically tries multiple time formats:

```go
type EventForm struct {
    StartDate time.Time `form:"start_date"`  // Accepts multiple formats
}

// Supported formats:
// - RFC3339: "2006-01-02T15:04:05Z07:00"
// - ISO8601: "2006-01-02T15:04:05"
// - DateTime: "2006-01-02 15:04:05"
// - Date: "2006-01-02"
// - Time: "15:04:05"
// - Compact: "20060102"
// - US Date: "02/01/2006"
```

### Mixed Form Data

Process forms with both files and text fields:

```go
type ProfileUpdateForm struct {
    Username  string                `form:"username,required"`
    Email     string                `form:"email,required"`
    Bio       string                `form:"bio,maxsize:500"`
    Avatar    *blaze.MultipartFile  `form:"avatar"`
    Settings  []string              `form:"settings"`
}

app.POST("/profile/update", func(c *blaze.Context) error {
    var form ProfileUpdateForm
    
    if err := c.BindMultipartFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    var avatarPath string
    if form.Avatar != nil {
        avatarPath, _ = form.Avatar.SaveWithUniqueFilename("./avatars")
    }
    
    return c.JSON(blaze.Map{
        "username":    form.Username,
        "email":       form.Email,
        "avatar_path": avatarPath,
        "settings":    form.Settings,
    })
})
```

## Multipart Configuration

### Default Configuration

Use default multipart settings:

```go
app.POST("/form-upload", func(c *blaze.Context) error {
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid form data"})
    }
    
    // Clean up temporary files when done
    defer form.Cleanup()
    
    // Access form data
    title := form.GetValue("title")
    tags := form.GetValues("tags")
    file := form.GetFile("upload")
    files := form.GetFiles("documents")
    
    return c.JSON(blaze.Map{
        "files_count": form.GetFileCount(),
        "total_size":  form.GetTotalSize(),
        "fields":      len(form.Value),
        "title":       title,
        "tags":        tags,
    })
})
```

### Custom Configuration

Configure multipart parsing with custom settings:

```go
app.POST("/custom-upload", func(c *blaze.Context) error {
    config := blaze.MultipartConfig{
        MaxMemory:   10 * 1024 * 1024,  // 10MB
        MaxFileSize: 50 * 1024 * 1024,  // 50MB per file
        MaxFiles:    5,                  // Maximum 5 files
        AllowedExtensions: []string{".jpg", ".png", ".pdf"},
        AllowedMimeTypes: []string{
            "image/jpeg",
            "image/png",
            "application/pdf",
        },
        KeepInMemory: false,  // Save large files to disk
        AutoCleanup:  true,   // Auto cleanup temp files
        TempDir:      "./temp",
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    defer form.Cleanup()
    
    return c.JSON(blaze.Map{"message": "Form processed successfully"})
})
```

### Production Configuration

Use production-ready multipart settings:

```go
func setupProductionUpload(app *blaze.App) {
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
                
                // Validate size
                if file.Size > 50*1024*1024 {
                    continue
                }
                
                // Save with unique filename
                savePath, err := file.SaveWithUniqueFilename("./secure_uploads")
                if err != nil {
                    continue
                }
                
                log.Printf("Saved file: %s from field: %s", savePath, fieldName)
            }
        }
        
        return c.JSON(blaze.Map{"message": "Files processed successfully"})
    })
}
```

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `MaxMemory` | `int64` | Max memory for parsing (bytes) | `32MB` |
| `MaxFileSize` | `int64` | Max size per file (bytes) | `100MB` |
| `MaxFiles` | `int` | Maximum number of files | `10` |
| `TempDir` | `string` | Temp directory for large files | OS temp |
| `AllowedExtensions` | `[]string` | Allowed file extensions | All |
| `AllowedMimeTypes` | `[]string` | Allowed MIME types | All |
| `KeepInMemory` | `bool` | Keep files in memory | `true` |
| `AutoCleanup` | `bool` | Auto cleanup temp files | `true` |

### MultipartForm Methods

```go
// Value access
value := form.GetValue("field")           // Single value
values := form.GetValues("field")         // Multiple values
hasValue := form.HasValue("field")        // Check existence

// File access
file := form.GetFile("field")             // Single file
files := form.GetFiles("field")           // Multiple files
hasFile := form.HasFile("field")          // Check existence

// Statistics
totalSize := form.GetTotalSize()          // Total size in bytes
fileCount := form.GetFileCount()          // Number of files

// Cleanup
err := form.Cleanup()                     // Remove temp files
```

## File Validation

### Size Limit Middleware

Restrict upload file sizes globally:

```go
// Limit all uploads to 10MB
app.Use(blaze.BodyLimitMB(10))

app.POST("/upload", func(c *blaze.Context) error {
    // File size already validated by middleware
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    return c.JSON(blaze.Map{"message": "File within size limits"})
})
```

### Manual Validation

Validate files manually in handlers:

```go
func validateUploadedFile(file *blaze.MultipartFile) error {
    // Size validation
    maxSize := int64(10 * 1024 * 1024) // 10MB
    if file.Size > maxSize {
        return fmt.Errorf("file too large (max 10MB)")
    }
    
    if file.Size == 0 {
        return fmt.Errorf("empty file not allowed")
    }
    
    // Type validation
    allowedTypes := []string{"image/jpeg", "image/png", "application/pdf"}
    allowed := false
    for _, t := range allowedTypes {
        if file.ContentType == t {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("file type not allowed: %s", file.ContentType)
    }
    
    // Extension validation
    allowedExtensions := []string{".jpg", ".jpeg", ".png", ".pdf"}
    ext := file.GetExtension()
    allowed = false
    for _, e := range allowedExtensions {
        if strings.EqualFold(ext, e) {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("file extension not allowed: %s", ext)
    }
    
    // Filename validation (security)
    if strings.Contains(file.Filename, "..") || 
       strings.Contains(file.Filename, "/") ||
       strings.Contains(file.Filename, "\\") {
        return fmt.Errorf("invalid filename")
    }
    
    return nil
}

app.POST("/upload-validated", func(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file provided"})
    }
    
    if err := validateUploadedFile(file); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    savePath, _ := file.SaveWithUniqueFilename("./uploads")
    
    return c.JSON(blaze.Map{
        "message": "File validated and uploaded",
        "path":    savePath,
    })
})
```

### Body Size Validation

Check request body size before processing:

```go
app.POST("/upload-large", func(c *blaze.Context) error {
    // Validate body size (50MB max)
    if err := c.ValidateBodySize(50 * 1024 * 1024); err != nil {
        return c.Status(413).JSON(blaze.Map{
            "error": "Request too large",
            "details": err.Error(),
        })
    }
    
    // Get body size info
    bodySize := c.GetBodySize()
    contentLength := c.GetContentLength()
    
    // Process upload...
    
    return c.JSON(blaze.Map{
        "body_size":      bodySize,
        "content_length": contentLength,
    })
})
```

## Static File Serving

### Basic Static File Handler

Create a static file serving handler:

```go
func StaticFileHandler(directory string) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := fmt.Sprintf("%s/%s", directory, filename)
        
        // Security check - prevent directory traversal
        if strings.Contains(filename, "..") || 
           strings.Contains(filename, "/") {
            return c.Status(403).JSON(blaze.Map{
                "error": "Access denied",
            })
        }
        
        if !c.FileExists(filepath) {
            return c.Status(404).JSON(blaze.Map{
                "error": "File not found",
            })
        }
        
        return c.ServeFileInline(filepath)
    }
}

// Usage
app.GET("/static/:filename", StaticFileHandler("./public"))
app.GET("/images/:filename", StaticFileHandler("./uploads/images"))
app.GET("/downloads/:filename", StaticFileHandler("./downloads"))
```

### Static File Configuration

Use the built-in static file serving with configuration:

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Index = "index.html"
staticConfig.Browse = false  // Disable directory browsing
staticConfig.Compress = true
staticConfig.CacheDuration = 24 * time.Hour
staticConfig.GenerateETag = true
staticConfig.ByteRange = true  // Enable range requests

app.Use("/static", blaze.StaticFS(staticConfig))
```

## Best Practices

### Security Considerations

1. **File Type Validation**: Always validate both extension and MIME type
2. **File Size Limits**: Implement appropriate size limits to prevent DoS
3. **Filename Sanitization**: Never trust user-provided filenames
4. **Directory Traversal Protection**: Block access outside designated directories
5. **Virus Scanning**: Consider integrating virus scanning for uploads
6. **Content Inspection**: Verify file contents match declared type

```go
func secureFileUpload(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file"})
    }
    
    // Comprehensive validation
    if err := validateUploadedFile(file); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Use unique filename to prevent conflicts/overwrites
    savePath, err := file.SaveWithUniqueFilename("./secure_uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Save failed"})
    }
    
    // Log upload for audit
    log.Printf("File uploaded: %s by IP: %s", savePath, c.IP())
    
    return c.JSON(blaze.Map{"path": savePath})
}
```

### Performance Optimization

1. **Use Streaming**: For large files, use `StreamFile` to reduce memory
2. **Implement Caching**: Cache frequently accessed files
3. **Compression**: Use compression middleware for text-based files
4. **CDN Integration**: Serve static files from a CDN
5. **Unique Filenames**: Prevent naming conflicts with timestamp-based names

```go
// Efficient large file upload
app.POST("/upload-large", func(c *blaze.Context) error {
    config := blaze.MultipartConfig{
        MaxMemory:    10 * 1024 * 1024,  // Only 10MB in memory
        KeepInMemory: false,              // Save to disk
        TempDir:      "./temp",
    }
    
    form, err := c.MultipartFormWithConfig(config)
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    defer form.Cleanup()
    
    // Process files...
    return c.JSON(blaze.Map{"message": "Success"})
})
```

### Error Handling

Always implement comprehensive error handling:

```go
app.POST("/robust-upload", func(c *blaze.Context) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Upload panic: %v", r)
            c.Status(500).JSON(blaze.Map{"error": "Upload failed"})
        }
    }()
    
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No file uploaded",
            "code":  "NO_FILE",
        })
    }
    
    if file.Size > 10*1024*1024 {
        return c.Status(413).JSON(blaze.Map{
            "error": "File too large",
            "code":  "FILE_TOO_LARGE",
            "limit": "10MB",
            "size":  file.Size,
        })
    }
    
    savePath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
    if err != nil {
        log.Printf("Save failed for %s: %v", file.Filename, err)
        return c.Status(500).JSON(blaze.Map{
            "error": "Failed to save file",
            "code":  "SAVE_FAILED",
        })
    }
    
    return c.JSON(blaze.Map{
        "message":   "Success",
        "filename":  file.Filename,
        "save_path": savePath,
        "size":      file.Size,
    })
})
```

This comprehensive file handling guide covers all aspects of working with files in Blaze, from basic operations to advanced struct binding with complete validation support.