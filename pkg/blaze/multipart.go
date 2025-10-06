package blaze

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MultipartFile represents an uploaded file with metadata and data
// Provides comprehensive file information and utility methods
//
// File Storage:
//   - Small files: Kept in memory (Data field)
//   - Large files: Saved to temporary files (TempFilePath field)
//   - Configurable threshold via MaxMemory in MultipartConfig
//
// File Access:
//   - Data: In-memory file content (for small files)
//   - TempFilePath: Path to temporary file (for large files)
//   - FileHeader: Original multipart.FileHeader from HTTP request
//
// Common Operations:
//   - Save(): Save file to specific path
//   - SaveToDir(): Save with original filename to directory
//   - SaveWithUniqueFilename(): Save with auto-generated unique name
//   - Cleanup(): Remove temporary files
//
// Security Considerations:
//   - Always sanitize filenames before saving
//   - Validate file types and extensions
//   - Implement file size limits
//   - Store uploads outside web root
//   - Scan for viruses in production
type MultipartFile struct {
	// Filename is the original filename from the upload
	// May contain path separators or unsafe characters
	// Use sanitizeFilename() before using in filesystem operations
	Filename string

	// Header contains all HTTP headers from the multipart section
	// Includes Content-Type, Content-Disposition, custom headers
	// Access via: file.Header["Content-Type"]
	Header map[string][]string

	// Size is the file size in bytes
	// Validated against MaxFileSize in MultipartConfig
	Size int64

	// ContentType is the MIME type from Content-Type header
	// Examples: "image/jpeg", "application/pdf", "text/plain"
	// Can be spoofed by clients - validate file content for security
	ContentType string

	// Data contains the file content when kept in memory
	// Populated when file size <= MaxMemory
	// Empty when file is saved to temporary file
	Data []byte

	// TempFilePath is the path to temporary file for large uploads
	// Set when file size > MaxMemory
	// Empty when file is kept in memory
	// Automatically cleaned up if AutoCleanup is enabled
	TempFilePath string

	// FileHeader is the original multipart.FileHeader
	// Provides access to low-level multipart functionality
	// Can be used to re-open the file or access additional metadata
	FileHeader *multipart.FileHeader
}

// MultipartForm represents parsed multipart form data
// Contains both form fields (text values) and uploaded files
//
// Form Structure:
//   - Value: Text form fields (input, textarea, select)
//   - File: Uploaded files organized by field name
//
// Usage Pattern:
//
//	form, err := c.MultipartForm()
//	name := form.GetValue("name")
//	file := form.GetFile("avatar")
//
// Multiple Values:
//   - Use GetValues() for fields with multiple values (checkboxes, multi-select)
//   - Use GetFiles() for multiple file uploads with same field name
type MultipartForm struct {
	// Value maps form field names to their values
	// Each field name maps to a slice of strings (supports multiple values)
	// Example: {"name": []string{"John"}, "tags": []string{"go", "web"}}
	Value map[string][]string

	// File maps form field names to uploaded files
	// Each field name maps to a slice of MultipartFile (supports multiple files)
	// Example: {"avatar": []MultipartFile{...}, "documents": []MultipartFile{...}}
	File map[string][]*MultipartFile
}

// MultipartConfig holds multipart form parsing and validation configuration
// Controls memory usage, file size limits, type restrictions, and cleanup behavior
//
// Configuration Philosophy:
//   - Development: Permissive limits, keep in memory, auto cleanup
//   - Production: Strict limits, disk storage, validated types, auto cleanup
//
// Memory Management:
//   - MaxMemory: Files <= this size stay in memory
//   - Files > MaxMemory: Saved to TempDir
//   - Balance: Lower MaxMemory saves RAM, higher reduces disk I/O
//
// Security:
//   - Always set MaxFileSize to prevent DOS attacks
//   - Restrict AllowedExtensions and AllowedMimeTypes
//   - Use AutoCleanup to prevent disk space exhaustion
//   - Validate file content, not just extensions/MIME types
type MultipartConfig struct {
	// MaxMemory specifies maximum bytes to keep in memory per file
	// Files larger than this are saved to temporary files
	// Recommended: 10-32 MB for typical applications
	// Default: 32 MB (32 << 20)
	MaxMemory int64

	// MaxFileSize specifies maximum allowed file size in bytes
	// Uploads exceeding this limit are rejected
	// Prevents DOS attacks via large file uploads
	// Set to 0 for no limit (not recommended)
	// Recommended: 50-100 MB for general use, less for specific uses
	// Default: 100 MB (100 << 20)
	MaxFileSize int64

	// MaxFiles limits the number of files per request
	// Prevents DOS attacks via many small files
	// Set to 0 for no limit (not recommended)
	// Recommended: 5-10 files for typical applications
	// Default: 10
	MaxFiles int

	// TempDir specifies directory for temporary file storage
	// Used when file size exceeds MaxMemory
	// Must have sufficient disk space and proper permissions
	// Automatically cleaned if AutoCleanup is enabled
	// Default: os.TempDir() (system temp directory)
	TempDir string

	// AllowedExtensions restricts file extensions
	// Case-insensitive comparison
	// Empty list allows all extensions
	// Example: []string{".jpg", ".png", ".pdf"}
	// Security: Extensions can be spoofed, validate content
	AllowedExtensions []string

	// AllowedMimeTypes restricts MIME types
	// Checked against Content-Type header
	// Empty list allows all MIME types
	// Example: []string{"image/jpeg", "application/pdf"}
	// Security: MIME types can be spoofed, validate content
	AllowedMimeTypes []string

	// KeepInMemory forces all files to be kept in memory
	// When true, ignores MaxMemory and never saves to disk
	// Useful for testing or when files are very small
	// WARNING: Can cause OOM with large uploads
	// Default: true
	KeepInMemory bool

	// AutoCleanup enables automatic cleanup of temporary files
	// When true, temporary files are deleted after request completes
	// When false, caller is responsible for cleanup
	// Should always be true unless manual cleanup is needed
	// Default: true
	AutoCleanup bool
}

// DefaultMultipartConfig returns default multipart configuration
// Suitable for development with permissive limits
//
// Default Settings:
//   - MaxMemory: 32 MB
//   - MaxFileSize: 100 MB
//   - MaxFiles: 10
//   - TempDir: System temp directory
//   - AllowedExtensions: None (all allowed)
//   - AllowedMimeTypes: None (all allowed)
//   - KeepInMemory: true
//   - AutoCleanup: true
//
// Returns:
//   - MultipartConfig: Default configuration
func DefaultMultipartConfig() *MultipartConfig {
	return &MultipartConfig{
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

// ProductionMultipartConfig returns production-ready multipart configuration
// Implements strict limits and type restrictions for security
//
// Production Settings:
//   - MaxMemory: 10 MB (lower for better memory management)
//   - MaxFileSize: 50 MB (prevents large uploads)
//   - MaxFiles: 5 (limits concurrent uploads)
//   - TempDir: /tmp/uploads (dedicated directory)
//   - AllowedExtensions: Common safe types only
//   - AllowedMimeTypes: Validated MIME types
//   - KeepInMemory: false (uses disk for large files)
//   - AutoCleanup: true (prevents disk space issues)
//
// Allowed File Types:
//   - Images: .jpg, .jpeg, .png, .gif
//   - Documents: .pdf, .txt, .csv, .doc, .docx, .xls, .xlsx
//
// Returns:
//   - MultipartConfig: Production-ready configuration
func ProductionMultipartConfig() *MultipartConfig {
	return &MultipartConfig{
		MaxMemory:   10 << 20, // 10 MB
		MaxFileSize: 50 << 20, // 50 MB
		MaxFiles:    5,
		TempDir:     "/tmp/uploads",
		AllowedExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif", ".pdf",
			".txt", ".csv", ".doc", ".docx", ".xls", ".xlsx",
		},
		AllowedMimeTypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"application/pdf", "text/plain", "text/csv",
			"application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		KeepInMemory: false,
		AutoCleanup:  true,
	}
}

// Save saves the uploaded file to the specified path
// Creates parent directories if they don't exist
//
// File Source Priority:
//  1. In-memory data (Data field)
//  2. Temporary file (TempFilePath field)
//  3. Error if neither available
//
// Parameters:
//   - path: Destination file path
//
// Returns:
//   - error: Save error or nil on success
//
// Example:
//
//	file, _ := c.FormFile("avatar")
//	err := file.Save("/uploads/avatar.jpg")
func (f *MultipartFile) Save(path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// If data is in memory, write it directly
	if len(f.Data) > 0 {
		return os.WriteFile(path, f.Data, 0644)
	}

	// If data is in temp file, move it
	if f.TempFilePath != "" {
		return os.Rename(f.TempFilePath, path)
	}

	return fmt.Errorf("no file data available")
}

// SaveToDir saves the file to a directory with the original filename
// Uses the filename from the upload, sanitized for safety
//
// Sanitization:
//   - Removes path separators (/, \)
//   - Removes parent directory references (..)
//   - Trims leading dots
//   - Falls back to "upload" if filename is invalid
//
// Parameters:
//   - dir: Destination directory
//
// Returns:
//   - string: Full path where file was saved
//   - error: Save error or nil on success
//
// Example:
//
//	file, _ := c.FormFile("document")
//	path, err := file.SaveToDir("/uploads/documents")
//	// Saves to: /uploads/documents/original-filename.pdf
func (f *MultipartFile) SaveToDir(dir string) (string, error) {
	// Sanitize filename
	filename := sanitizeFilename(f.Filename)
	if filename == "" {
		filename = fmt.Sprintf("upload_%d", time.Now().UnixNano())
	}

	path := filepath.Join(dir, filename)
	err := f.Save(path)
	return path, err
}

// SaveWithUniqueFilename saves the file with a unique generated filename
// Prevents filename collisions by adding timestamp to filename
//
// Filename Format:
//   - Original: "document.pdf"
//   - Generated: "document_1609459200000000000.pdf"
//   - Pattern: {base}_{timestamp}{extension}
//
// Use Cases:
//   - Preventing overwrites
//   - Multi-user uploads
//   - Versioning
//   - Concurrent uploads
//
// Parameters:
//   - dir: Destination directory
//
// Returns:
//   - string: Full path where file was saved
//   - error: Save error or nil on success
//
// Example:
//
//	file, _ := c.FormFile("attachment")
//	path, err := file.SaveWithUniqueFilename("/uploads")
//	// Saves to: /uploads/attachment_1609459200000000000.pdf
func (f *MultipartFile) SaveWithUniqueFilename(dir string) (string, error) {
	// Generate unique filename
	ext := filepath.Ext(f.Filename)
	base := strings.TrimSuffix(filepath.Base(f.Filename), ext)
	base = sanitizeFilename(base)

	if base == "" {
		base = "upload"
	}

	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("%s_%d%s", base, timestamp, ext)
	path := filepath.Join(dir, filename)

	err := f.Save(path)
	return path, err
}

// GetExtension returns the file extension in lowercase
// Includes the leading dot
//
// Returns:
//   - string: File extension (e.g., ".jpg", ".pdf")
//
// Example:
//
//	file.Filename = "document.PDF"
//	ext := file.GetExtension() // Returns: ".pdf"
func (f *MultipartFile) GetExtension() string {
	return strings.ToLower(filepath.Ext(f.Filename))
}

// GetMimeType returns the MIME type from Content-Type header
// Returns the MIME type as provided by the client
//
// Security Warning:
//   - MIME types can be spoofed by clients
//   - Use for convenience, not security decisions
//   - Validate actual file content for security-critical applications
//
// Returns:
//   - string: MIME type (e.g., "image/jpeg", "application/pdf")
func (f *MultipartFile) GetMimeType() string {
	return f.ContentType
}

// IsImage checks if the file is an image based on MIME type
// Checks if MIME type starts with "image/"
//
// Supported Image Types:
//   - image/jpeg, image/png, image/gif
//   - image/bmp, image/webp, image/svg+xml
//   - Any other image/* MIME type
//
// Returns:
//   - bool: true if file is an image
//
// Example:
//
//	if file.IsImage() {
//	    // Process as image
//	}
func (f *MultipartFile) IsImage() bool {
	mimeType := f.GetMimeType()
	return strings.HasPrefix(mimeType, "image/")
}

// IsDocument checks if the file is a document
// Checks MIME type for common document formats
//
// Supported Document Types:
//   - application/*: PDF, Word, Excel, etc.
//   - text/*: Plain text, CSV, etc.
//
// Returns:
//   - bool: true if file is a document
//
// Example:
//
//	if file.IsDocument() {
//	    // Process as document
//	}
func (f *MultipartFile) IsDocument() bool {
	mimeType := f.GetMimeType()
	return strings.HasPrefix(mimeType, "application/") ||
		strings.HasPrefix(mimeType, "text/")
}

// Cleanup removes temporary files if they exist
// Should be called when file is no longer needed
// Automatically called if AutoCleanup is enabled
//
// Returns:
//   - error: Removal error or nil on success
//
// Example:
//
//	defer file.Cleanup()
func (f *MultipartFile) Cleanup() error {
	if f.TempFilePath != "" {
		return os.Remove(f.TempFilePath)
	}
	return nil
}

// sanitizeFilename removes potentially dangerous characters from filename
// Prevents directory traversal and other filesystem attacks
//
// Sanitization Steps:
//  1. Remove path separators (/ and \)
//  2. Remove parent directory references (..)
//  3. Trim whitespace
//  4. Remove leading dots
//
// Parameters:
//   - filename: Original filename to sanitize
//
// Returns:
//   - string: Sanitized filename safe for filesystem use
func sanitizeFilename(filename string) string {
	// Remove path separators and other dangerous characters
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	filename = strings.TrimSpace(filename)

	// Remove leading dots
	filename = strings.TrimLeft(filename, ".")

	return filename
}

// validateFile checks if the file meets the configuration requirements
// Validates file size, extension, and MIME type
//
// Validation Rules:
//  1. File size must not exceed MaxFileSize
//  2. Extension must be in AllowedExtensions (if specified)
//  3. MIME type must be in AllowedMimeTypes (if specified)
//
// Parameters:
//   - config: MultipartConfig with validation rules
//   - file: MultipartFile to validate
//
// Returns:
//   - error: Validation error or nil if valid
func (config *MultipartConfig) validateFile(file *MultipartFile) error {
	// Check file size
	if config.MaxFileSize > 0 && file.Size > config.MaxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", file.Size, config.MaxFileSize)
	}

	// Check file extension
	if len(config.AllowedExtensions) > 0 {
		ext := file.GetExtension()
		allowed := false
		for _, allowedExt := range config.AllowedExtensions {
			if strings.EqualFold(ext, allowedExt) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file extension %s is not allowed", ext)
		}
	}

	// Check MIME type
	if len(config.AllowedMimeTypes) > 0 {
		mimeType := file.GetMimeType()
		allowed := false
		for _, allowedType := range config.AllowedMimeTypes {
			if strings.EqualFold(mimeType, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("MIME type %s is not allowed", mimeType)
		}
	}

	return nil
}

// MultipartForm helper methods

// GetTotalSize returns the total size of all uploaded files in bytes
// Useful for monitoring and validation
//
// Returns:
//   - int64: Total size in bytes
func (f *MultipartForm) GetTotalSize() int64 {
	var total int64
	for _, files := range f.File {
		for _, file := range files {
			total += file.Size
		}
	}
	return total
}

// GetFileCount returns the total number of uploaded files
// Counts all files across all form fields
//
// Returns:
//   - int: Total number of files
func (f *MultipartForm) GetFileCount() int {
	var count int
	for _, files := range f.File {
		count += len(files)
	}
	return count
}

// Cleanup removes all temporary files from the form
// Should be called when form is no longer needed
// Automatically called if AutoCleanup is enabled
//
// Returns:
//   - error: Last error encountered or nil
func (f *MultipartForm) Cleanup() error {
	var lastError error
	for _, files := range f.File {
		for _, file := range files {
			if err := file.Cleanup(); err != nil {
				lastError = err
			}
		}
	}
	return lastError
}

// GetFile returns the first file for the given field name
// Returns nil if field doesn't exist or has no files
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - *MultipartFile: First uploaded file or nil
func (f *MultipartForm) GetFile(name string) *MultipartFile {
	if files, exists := f.File[name]; exists && len(files) > 0 {
		return files[0]
	}
	return nil
}

// GetFiles returns all files for the given field name
// Returns nil if field doesn't exist
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - []MultipartFile: All uploaded files or nil
func (f *MultipartForm) GetFiles(name string) []*MultipartFile {
	if files, exists := f.File[name]; exists {
		return files
	}
	return nil
}

// GetValue returns the first value for the given field name
// Returns empty string if field doesn't exist
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - string: First form value or empty string
func (f *MultipartForm) GetValue(name string) string {
	if values, exists := f.Value[name]; exists && len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetValues returns all values for the given field name
// Used for multi-value fields (checkboxes, multi-select)
// Returns nil if field doesn't exist
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - []string: All form values or nil
func (f *MultipartForm) GetValues(name string) []string {
	if values, exists := f.Value[name]; exists {
		return values
	}
	return nil
}

// HasFile checks if a file exists for the given field name
// Returns true if at least one file was uploaded for the field
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - bool: true if files exist
func (f *MultipartForm) HasFile(name string) bool {
	files, exists := f.File[name]
	return exists && len(files) > 0
}

// HasValue checks if a value exists for the given field name
// Returns true if at least one value exists for the field
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - bool: true if values exist
func (f *MultipartForm) HasValue(name string) bool {
	values, exists := f.Value[name]
	return exists && len(values) > 0
}
