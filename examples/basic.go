//go:build ignore

package main

import (
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
	// Custom configuration
	config := &blaze.Config{
		Host:               "0.0.0.0",
		Port:               3000,
		ReadTimeout:        15 * time.Second,
		WriteTimeout:       15 * time.Second,
		MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
		Concurrency:        512 * 1024,
		EnableHTTP2:        false,
		EnableTLS:          false,
	}

	app := blaze.NewWithConfig(config)

	// Global middleware
	app.Use(blaze.Recovery())
	// app.Use(blaze.Logger())
	app.Use(blaze.CORS(blaze.CORSOptions{
		AllowedOrigins:   []string{"http://localhost:3000", "https://yourdomain.com"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	// Routes
	setupRoutes(app)

	// Start server
	app.ListenAndServe()
}

func setupRoutes(app *blaze.App) {
	// Root endpoint
	app.GET("/", func(c *blaze.Context) error {
		return c.JSON(blaze.Map{
			"name":        "Blaze Framework",
			"version":     "1.0.0",
			"description": "A blazing fast Go web framework",
			"endpoints": blaze.Map{
				"health": "/health",
				"api":    "/api/v1",
				"docs":   "/docs",
			},
		})
	})

	// Health check
	app.GET("/health", func(c *blaze.Context) error {
		return c.JSON(blaze.Health("1.0.0", "running"))
	})

	// API routes
	api := app.Group("/api/v1")

	// Example CRUD endpoints
	api.GET("/items", listItems)
	api.POST("/items", createItem)
	api.GET("/items/:id", getItem, blaze.WithIntConstraint("id"))
	api.PUT("/items/:id", updateItem)
	api.DELETE("/items/:id", deleteItem)

	// File upload example
	api.POST("/upload", uploadFile)

	// WebSocket endpoint (placeholder)
	api.GET("/ws", websocketHandler)
}

type Item struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

var items = []Item{
	{
		ID:          1,
		Name:        "Laptop",
		Description: "High-performance laptop",
		Price:       999.99,
		Created:     time.Now(),
		Updated:     time.Now(),
	},
}

func listItems(c *blaze.Context) error {
	page := c.QueryIntDefault("page", 1)
	perPage := c.QueryIntDefault("per_page", 10)
	search := c.Query("search")

	filteredItems := items

	// Simple search
	if search != "" {
		filteredItems = []Item{}
		for _, item := range items {
			if contains(item.Name, search) || contains(item.Description, search) {
				filteredItems = append(filteredItems, item)
			}
		}
	}

	// Pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(filteredItems) {
		end = len(filteredItems)
	}
	if start > len(filteredItems) {
		start = len(filteredItems)
	}

	paginatedItems := filteredItems[start:end]
	response := blaze.Paginate(paginatedItems, len(filteredItems), page, perPage)

	return c.JSON(response)
}

func createItem(c *blaze.Context) error {
	var newItem struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}

	if err := c.BindJSON(&newItem); err != nil {
		return blaze.BadRequest(c, "Invalid request body")
	}

	if newItem.Name == "" {
		return blaze.BadRequest(c, "Name is required")
	}

	item := Item{
		ID:          len(items) + 1,
		Name:        newItem.Name,
		Description: newItem.Description,
		Price:       newItem.Price,
		Created:     time.Now(),
		Updated:     time.Now(),
	}

	items = append(items, item)

	return c.Status(201).JSON(blaze.Created(item))
}

func getItem(c *blaze.Context) error {
	id, err := c.ParamInt("id")

	if err != nil {
		return blaze.BadRequest(c, "Invalid ID format")
	}

	for _, item := range items {
		if item.ID == id {
			return c.JSON(blaze.OK(item))
		}
	}

	return blaze.NotFound(c, "Item not found")
}

func updateItem(c *blaze.Context) error {
	// Implementation similar to createItem but for updates
	return c.JSON(blaze.Map{"message": "Update functionality"})
}

func deleteItem(c *blaze.Context) error {
	// Implementation for deleting items
	return c.Status(204).Text("")
}

func uploadFile(c *blaze.Context) error {
	// File upload implementation
	return c.JSON(blaze.Map{"message": "File upload functionality"})
}

func websocketHandler(c *blaze.Context) error {
	// WebSocket implementation placeholder
	return c.JSON(blaze.Map{"message": "WebSocket endpoint"})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				s[:len(substr)] == substr))
}
