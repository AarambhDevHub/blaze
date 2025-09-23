//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	app := blaze.New()

	// Create upload directory
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatal("Failed to create upload directory:", err)
	}

	// Add middleware
	app.Use(blaze.Logger())
	app.Use(blaze.Recovery())
	app.Use(blaze.CORS())

	// Multipart middleware with custom config
	multipartConfig := blaze.ProductionMultipartConfig()
	multipartConfig.MaxFileSize = 10 << 20 // 10MB
	multipartConfig.MaxFiles = 3
	app.Use(blaze.MultipartMiddleware(multipartConfig))

	// Routes
	app.GET("/", func(c *blaze.Context) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Blaze Multipart Upload</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .upload-form { border: 2px dashed #ccc; padding: 30px; text-align: center; }
        .file-info { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        input, button { margin: 10px; padding: 10px; font-size: 14px; }
        button { background: #007bff; color: white; border: none; border-radius: 3px; cursor: pointer; }
        button:hover { background: #0056b3; }
    </style>
</head>
<body>
    <h1>🚀 Blaze Framework - Multipart File Upload</h1>

    <div class="upload-form">
        <h2>📁 Single File Upload</h2>
        <form action="/upload/single" method="post" enctype="multipart/form-data">
            <input type="text" name="description" placeholder="File description" required><br>
            <input type="file" name="file" required><br>
            <button type="submit">Upload Single File</button>
        </form>
    </div>

    <div class="upload-form">
        <h2>📁 Multiple Files Upload</h2>
        <form action="/upload/multiple" method="post" enctype="multipart/form-data">
            <input type="text" name="category" placeholder="Category" required><br>
            <input type="file" name="files" multiple required><br>
            <button type="submit">Upload Multiple Files</button>
        </form>
    </div>

    <div class="upload-form">
        <h2>🖼️ Image Upload Only</h2>
        <form action="/upload/images" method="post" enctype="multipart/form-data">
            <input type="text" name="album" placeholder="Album name"><br>
            <input type="file" name="images" accept="image/*" multiple><br>
            <button type="submit">Upload Images</button>
        </form>
    </div>

    <div class="upload-form">
        <h2>📄 Document Upload Only</h2>
        <form action="/upload/documents" method="post" enctype="multipart/form-data">
            <input type="text" name="project" placeholder="Project name"><br>
            <input type="file" name="documents" accept=".pdf,.doc,.docx,.txt" multiple><br>
            <button type="submit">Upload Documents</button>
        </form>
    </div>

    <div class="file-info">
        <h3>📋 API Endpoints</h3>
        <p><strong>POST /upload/single</strong> - Upload single file</p>
        <p><strong>POST /upload/multiple</strong> - Upload multiple files</p>
        <p><strong>POST /upload/images</strong> - Upload images only</p>
        <p><strong>POST /upload/documents</strong> - Upload documents only</p>
        <p><strong>GET /files</strong> - List uploaded files</p>
        <p><strong>GET /file/:filename</strong> - Download file</p>
    </div>
</body>
</html>`
		return c.HTML(html)
	})

	// Single file upload
	app.POST("/upload/single", func(c *blaze.Context) error {
		description := c.FormValue("description")

		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(blaze.Error("No file uploaded: " + err.Error()))
		}

		// Save with unique filename
		savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, uploadDir)
		if err != nil {
			return c.Status(500).JSON(blaze.Error("Failed to save file: " + err.Error()))
		}

		return c.JSON(blaze.Map{
			"message":     "File uploaded successfully",
			"description": description,
			"file": blaze.Map{
				"original_name": file.Filename,
				"saved_path":    savedPath,
				"size":          file.Size,
				"content_type":  file.ContentType,
				"extension":     file.GetExtension(),
			},
		})
	})

	// Multiple files upload
	app.POST("/upload/multiple", func(c *blaze.Context) error {
		category := c.FormValue("category")

		files, err := c.FormFiles("files")
		if err != nil {
			return c.Status(400).JSON(blaze.Error("No files uploaded: " + err.Error()))
		}

		var uploadedFiles []blaze.Map
		var totalSize int64

		for _, file := range files {
			savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, uploadDir)
			if err != nil {
				return c.Status(500).JSON(blaze.Error("Failed to save file " + file.Filename + ": " + err.Error()))
			}

			totalSize += file.Size
			uploadedFiles = append(uploadedFiles, blaze.Map{
				"original_name": file.Filename,
				"saved_path":    savedPath,
				"size":          file.Size,
				"content_type":  file.ContentType,
				"extension":     file.GetExtension(),
				"is_image":      file.IsImage(),
				"is_document":   file.IsDocument(),
			})
		}

		return c.JSON(blaze.Map{
			"message":        "Files uploaded successfully",
			"category":       category,
			"files_count":    len(files),
			"total_size":     totalSize,
			"uploaded_files": uploadedFiles,
		})
	})

	// Image upload with restriction
	imageGroup := app.Group("/upload")
	imageGroup.Use(blaze.ImageOnlyMiddleware())

	imageGroup.POST("/images", func(c *blaze.Context) error {
		album := c.FormValue("album")

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(blaze.Error("Failed to parse form: " + err.Error()))
		}

		images := form.GetFiles("images")
		if len(images) == 0 {
			return c.Status(400).JSON(blaze.Error("No images uploaded"))
		}

		// Create album directory
		albumDir := filepath.Join(uploadDir, "images", album)
		if err := os.MkdirAll(albumDir, 0755); err != nil {
			return c.Status(500).JSON(blaze.Error("Failed to create album directory"))
		}

		var uploadedImages []blaze.Map
		for _, image := range images {
			savedPath, err := image.SaveWithUniqueFilename(albumDir)
			if err != nil {
				return c.Status(500).JSON(blaze.Error("Failed to save image: " + err.Error()))
			}

			uploadedImages = append(uploadedImages, blaze.Map{
				"filename":     image.Filename,
				"saved_path":   savedPath,
				"size":         image.Size,
				"content_type": image.ContentType,
			})
		}

		return c.JSON(blaze.Map{
			"message":         "Images uploaded successfully",
			"album":           album,
			"images_count":    len(images),
			"uploaded_images": uploadedImages,
		})
	})

	// Document upload with restriction
	docGroup := app.Group("/upload")
	docGroup.Use(blaze.DocumentOnlyMiddleware())

	docGroup.POST("/documents", func(c *blaze.Context) error {
		project := c.FormValue("project")

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(blaze.Error("Failed to parse form: " + err.Error()))
		}

		documents := form.GetFiles("documents")
		if len(documents) == 0 {
			return c.Status(400).JSON(blaze.Error("No documents uploaded"))
		}

		// Create project directory
		projectDir := filepath.Join(uploadDir, "documents", project)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return c.Status(500).JSON(blaze.Error("Failed to create project directory"))
		}

		var uploadedDocs []blaze.Map
		for _, doc := range documents {
			savedPath, err := doc.SaveToDir(projectDir)
			if err != nil {
				return c.Status(500).JSON(blaze.Error("Failed to save document: " + err.Error()))
			}

			uploadedDocs = append(uploadedDocs, blaze.Map{
				"filename":     doc.Filename,
				"saved_path":   savedPath,
				"size":         doc.Size,
				"content_type": doc.ContentType,
				"extension":    doc.GetExtension(),
			})
		}

		return c.JSON(blaze.Map{
			"message":            "Documents uploaded successfully",
			"project":            project,
			"documents_count":    len(documents),
			"uploaded_documents": uploadedDocs,
		})
	})

	// List uploaded files
	app.GET("/files", func(c *blaze.Context) error {
		var fileList []blaze.Map

		err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(uploadDir, path)
			fileList = append(fileList, blaze.Map{
				"name":         info.Name(),
				"path":         relPath,
				"size":         info.Size(),
				"modified":     info.ModTime(),
				"download_url": "/file/" + filepath.Base(path),
			})
			return nil
		})

		if err != nil {
			return c.Status(500).JSON(blaze.Error("Failed to list files"))
		}

		return c.JSON(blaze.Map{
			"message":     "Files listed successfully",
			"files_count": len(fileList),
			"files":       fileList,
		})
	})

	// Download file
	app.GET("/file/:filename", func(c *blaze.Context) error {
		filename := c.Param("filename")

		// Find file in upload directory
		var filePath string
		err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.Name() == filename {
				filePath = path
				return filepath.SkipDir
			}
			return nil
		})

		if err != nil || filePath == "" {
			return c.Status(404).JSON(blaze.Error("File not found"))
		}

		// Set download headers
		c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.SetHeader("Content-Type", "application/octet-stream")

		return c.SendFile(filePath)
	})

	// API info endpoint
	app.GET("/api/info", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"name":        "Blaze Multipart Upload API",
			"version":     "1.0.0",
			"description": "Complete multipart form data handling with file uploads",
			"features": []string{
				"Single and multiple file uploads",
				"File type restrictions",
				"File size limits",
				"Automatic cleanup",
				"Image and document filtering",
				"Unique filename generation",
				"Directory organization",
			},
			"config": blaze.Map{
				"max_file_size": fmt.Sprintf("%d MB", multipartConfig.MaxFileSize/(1024*1024)),
				"max_files":     multipartConfig.MaxFiles,
				"upload_dir":    uploadDir,
			},
		})
	})

	log.Printf("🚀 Blaze Multipart Upload Server starting on http://localhost:8080")
	log.Printf("📁 Upload directory: %s", uploadDir)
	log.Printf("🔗 Open http://localhost:8080 in your browser")
	log.Fatal(app.ListenAndServe())
}
