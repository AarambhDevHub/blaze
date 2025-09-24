package blaze

import "time"

// Response helpers

// OK returns a standard OK response
func OK(data interface{}) Map {
	return Map{
		"success": true,
		"data":    data,
	}
}

// Error returns a standard error response
func Error(message string) Map {
	return Map{
		"success": false,
		"error":   message,
	}
}

// Created returns a standard created response
func Created(data interface{}) Map {
	return Map{
		"success": true,
		"data":    data,
		"message": "Resource created successfully",
	}
}

// NoContent returns an empty response with 204 status
func NoContent(c *Context) error {
	return c.Status(204).Text("")
}

// BadRequest returns a 400 bad request response
func BadRequest(c *Context, message string) error {
	return c.Status(400).JSON(Error(message))
}

// Unauthorized returns a 401 unauthorized response
func Unauthorized(c *Context, message string) error {
	return c.Status(401).JSON(Error(message))
}

// Forbidden returns a 403 forbidden response
func Forbidden(c *Context, message string) error {
	return c.Status(403).JSON(Error(message))
}

// NotFound returns a 404 not found response
func NotFound(c *Context, message string) error {
	return c.Status(404).JSON(Error(message))
}

// InternalServerError returns a 500 internal server error response
func InternalServerError(c *Context, message string) error {
	return c.Status(500).JSON(Error(message))
}

// Redirect function sends an HTTP redirect to the given URL with the specified status code
func Redirect(c Context, url string, statusCode int) error {
	c.SetHeader("Location", url)
	return c.Status(statusCode).Text("") // empty body is common for redirects
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// Paginate creates a paginated response
func Paginate(data interface{}, total, page, perPage int) *PaginatedResponse {
	totalPages := (total + perPage - 1) / perPage

	return &PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// HealthCheck response
type HealthCheck struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// Health returns a health check response
func Health(version, uptime string) *HealthCheck {
	return &HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   version,
		Uptime:    uptime,
	}
}
