package blaze

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MultipartFile represents an uploaded file
type MultipartFile struct {
	Filename     string
	Header       map[string][]string
	Size         int64
	ContentType  string
	Data         []byte
	TempFilePath string
	FileHeader   *multipart.FileHeader
}

// MultipartForm represents multipart form data
type MultipartForm struct {
	Value map[string][]string
	File  map[string][]*MultipartFile
}

// MultipartConfig holds multipart configuration
type MultipartConfig struct {
	// Maximum memory to use for parsing multipart form (in bytes)
	MaxMemory int64

	// Maximum file size allowed (in bytes)
	MaxFileSize int64

	// Maximum number of files
	MaxFiles int

	// Temporary directory for large files
	TempDir string

	// Allowed file extensions (if empty, all are allowed)
	AllowedExtensions []string

	// Allowed MIME types (if empty, all are allowed)
	AllowedMimeTypes []string

	// Whether to keep uploaded files in memory or save to disk
	KeepInMemory bool

	// Whether to automatically clean up temp files
	AutoCleanup bool
}

// DefaultMultipartConfig returns default multipart configuration
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

// SaveWithUniqueFilename saves the file with a unique filename to avoid conflicts
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

// GetExtension returns the file extension
func (f *MultipartFile) GetExtension() string {
	return strings.ToLower(filepath.Ext(f.Filename))
}

// GetMimeType returns the MIME type from Content-Type header
func (f *MultipartFile) GetMimeType() string {
	return f.ContentType
}

// IsImage checks if the file is an image
func (f *MultipartFile) IsImage() bool {
	mimeType := f.GetMimeType()
	return strings.HasPrefix(mimeType, "image/")
}

// IsDocument checks if the file is a document
func (f *MultipartFile) IsDocument() bool {
	mimeType := f.GetMimeType()
	return strings.HasPrefix(mimeType, "application/") ||
		strings.HasPrefix(mimeType, "text/")
}

// Cleanup removes temporary files if they exist
func (f *MultipartFile) Cleanup() error {
	if f.TempFilePath != "" {
		return os.Remove(f.TempFilePath)
	}
	return nil
}

// sanitizeFilename removes potentially dangerous characters from filename
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

// GetTotalSize returns the total size of all uploaded files
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
func (f *MultipartForm) GetFileCount() int {
	var count int
	for _, files := range f.File {
		count += len(files)
	}
	return count
}

// Cleanup removes all temporary files
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
func (f *MultipartForm) GetFile(name string) *MultipartFile {
	if files, exists := f.File[name]; exists && len(files) > 0 {
		return files[0]
	}
	return nil
}

// GetFiles returns all files for the given field name
func (f *MultipartForm) GetFiles(name string) []*MultipartFile {
	if files, exists := f.File[name]; exists {
		return files
	}
	return nil
}

// GetValue returns the first value for the given field name
func (f *MultipartForm) GetValue(name string) string {
	if values, exists := f.Value[name]; exists && len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetValues returns all values for the given field name
func (f *MultipartForm) GetValues(name string) []string {
	if values, exists := f.Value[name]; exists {
		return values
	}
	return nil
}

// HasFile checks if a file exists for the given field name
func (f *MultipartForm) HasFile(name string) bool {
	files, exists := f.File[name]
	return exists && len(files) > 0
}

// HasValue checks if a value exists for the given field name
func (f *MultipartForm) HasValue(name string) bool {
	values, exists := f.Value[name]
	return exists && len(values) > 0
}
