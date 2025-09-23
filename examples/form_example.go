//go:build ignore

package main

import (
	"log"
	"strings"
	"time"

	"github.com/AarambhDevHub/blaze/pkg/blaze"
)

// Example structs for form binding

// UserProfile represents a user profile form
type UserProfile struct {
	Name     string               `form:"name,required"`
	Email    string               `form:"email,required"`
	Age      int                  `form:"age,required"`
	Bio      string               `form:"bio,maxsize=500"`
	Website  string               `form:"website"`
	Avatar   *blaze.MultipartFile `form:"avatar"`
	IsActive bool                 `form:"is_active"`
	Score    float64              `form:"score"`
	Tags     []string             `form:"tags"`
	JoinedAt time.Time            `form:"joined_at"`
}

// ProductForm represents a product creation form
type ProductForm struct {
	Name        string                 `form:"name,required,maxsize=100"`
	Description string                 `form:"description,required,maxsize=1000"`
	Price       float64                `form:"price,required"`
	Category    string                 `form:"category,required"`
	InStock     bool                   `form:"in_stock"`
	Images      []*blaze.MultipartFile `form:"images"`
	Tags        []string               `form:"tags"`
	LaunchDate  *time.Time             `form:"launch_date"`
	Weight      *float64               `form:"weight"`
}

// BlogPost represents a blog post form
type BlogPost struct {
	Title       string                 `form:"title,required,maxsize=200"`
	Content     string                 `form:"content,required,minsize=10"`
	Author      string                 `form:"author,required"`
	Published   bool                   `form:"published"`
	PublishDate time.Time              `form:"publish_date"`
	Categories  []string               `form:"categories"`
	FeaturedImg *blaze.MultipartFile   `form:"featured_image"`
	Attachments []*blaze.MultipartFile `form:"attachments"`
	Views       int                    `form:"views,default=0"`
	Rating      float32                `form:"rating,default=0.0"`
}

// NestedExample demonstrates nested struct binding
type Address struct {
	Street  string `form:"street,required"`
	City    string `form:"city,required"`
	State   string `form:"state,required"`
	ZipCode string `form:"zip_code,required"`
	Country string `form:"country,default=USA"`
}

type UserWithAddress struct {
	Name    string               `form:"name,required"`
	Email   string               `form:"email,required"`
	Address Address              `form:"address"`
	Photo   *blaze.MultipartFile `form:"photo"`
}

func main() {
	app := blaze.New()

	// Add middleware
	app.Use(blaze.Logger())
	app.Use(blaze.Recovery())
	app.Use(blaze.MultipartMiddleware(nil))

	// Main page with forms
	app.GET("/", func(c *blaze.Context) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Form Binding Examples</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1000px; margin: 20px auto; padding: 20px; }
        .form-section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .form-section h2 { margin-top: 0; color: #333; }
        input, textarea, select { margin: 5px; padding: 8px; font-size: 14px; width: 300px; }
        button { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 3px; cursor: pointer; }
        button:hover { background: #0056b3; }
        .checkbox { width: auto; }
    </style>
</head>
<body>
    <h1>🚀 Blaze Framework - Form Binding Examples</h1>

    <div class="form-section">
        <h2>👤 User Profile Form</h2>
        <form action="/user-profile" method="post" enctype="multipart/form-data">
            <input type="text" name="name" placeholder="Name" required><br>
            <input type="email" name="email" placeholder="Email" required><br>
            <input type="number" name="age" placeholder="Age" required><br>
            <textarea name="bio" placeholder="Bio (max 500 chars)"></textarea><br>
            <input type="url" name="website" placeholder="Website"><br>
            <input type="file" name="avatar" accept="image/*"><br>
            <label><input type="checkbox" name="is_active" class="checkbox"> Is Active</label><br>
            <input type="number" step="0.01" name="score" placeholder="Score"><br>
            <input type="text" name="tags" placeholder="Tags (comma-separated)"><br>
            <input type="datetime-local" name="joined_at"><br>
            <button type="submit">Submit User Profile</button>
        </form>
    </div>

    <div class="form-section">
        <h2>🛍️ Product Form</h2>
        <form action="/product" method="post" enctype="multipart/form-data">
            <input type="text" name="name" placeholder="Product Name" required><br>
            <textarea name="description" placeholder="Description" required></textarea><br>
            <input type="number" step="0.01" name="price" placeholder="Price" required><br>
            <input type="text" name="category" placeholder="Category" required><br>
            <label><input type="checkbox" name="in_stock" class="checkbox"> In Stock</label><br>
            <input type="file" name="images" accept="image/*" multiple><br>
            <input type="text" name="tags" placeholder="Tags (comma-separated)"><br>
            <input type="date" name="launch_date"><br>
            <input type="number" step="0.01" name="weight" placeholder="Weight (kg)"><br>
            <button type="submit">Submit Product</button>
        </form>
    </div>

    <div class="form-section">
        <h2>📝 Blog Post Form</h2>
        <form action="/blog-post" method="post" enctype="multipart/form-data">
            <input type="text" name="title" placeholder="Title" required><br>
            <textarea name="content" placeholder="Content (minimum 10 characters)" required></textarea><br>
            <input type="text" name="author" placeholder="Author" required><br>
            <label><input type="checkbox" name="published" class="checkbox"> Published</label><br>
            <input type="datetime-local" name="publish_date"><br>
            <input type="text" name="categories" placeholder="Categories (comma-separated)"><br>
            <input type="file" name="featured_image" accept="image/*"><br>
            <input type="file" name="attachments" multiple><br>
            <input type="number" name="views" placeholder="Views"><br>
            <input type="number" step="0.1" name="rating" placeholder="Rating (0-5)"><br>
            <button type="submit">Submit Blog Post</button>
        </form>
    </div>

</body>
</html>`
		return c.HTML(html)
	})

	// User Profile endpoint
	app.POST("/user-profile", func(c *blaze.Context) error {
		var profile UserProfile

		if err := c.BindMultipartForm(&profile); err != nil {
			return c.Status(400).JSON(blaze.Map{
				"error":   "Form binding failed",
				"details": err.Error(),
			})
		}

		// Process tags (split comma-separated string into slice)
		if tagsStr := c.FormValue("tags"); tagsStr != "" {
			profile.Tags = strings.Split(tagsStr, ",")
			for i := range profile.Tags {
				profile.Tags[i] = strings.TrimSpace(profile.Tags[i])
			}
		}

		response := blaze.Map{
			"message": "User profile created successfully",
			"profile": blaze.Map{
				"name":      profile.Name,
				"email":     profile.Email,
				"age":       profile.Age,
				"bio":       profile.Bio,
				"website":   profile.Website,
				"is_active": profile.IsActive,
				"score":     profile.Score,
				"tags":      profile.Tags,
				"joined_at": profile.JoinedAt,
			},
		}

		// Add avatar info if present
		if profile.Avatar != nil {
			response["profile"].(blaze.Map)["avatar"] = blaze.Map{
				"filename":     profile.Avatar.Filename,
				"size":         profile.Avatar.Size,
				"content_type": profile.Avatar.ContentType,
			}
		}

		return c.JSON(response)
	})

	// Product endpoint
	app.POST("/product", func(c *blaze.Context) error {
		var product ProductForm

		if err := c.BindMultipartForm(&product); err != nil {
			return c.Status(400).JSON(blaze.Map{
				"error":   "Form binding failed",
				"details": err.Error(),
			})
		}

		// Process tags
		if tagsStr := c.FormValue("tags"); tagsStr != "" {
			product.Tags = strings.Split(tagsStr, ",")
			for i := range product.Tags {
				product.Tags[i] = strings.TrimSpace(product.Tags[i])
			}
		}

		response := blaze.Map{
			"message": "Product created successfully",
			"product": blaze.Map{
				"name":        product.Name,
				"description": product.Description,
				"price":       product.Price,
				"category":    product.Category,
				"in_stock":    product.InStock,
				"tags":        product.Tags,
				"weight":      product.Weight,
			},
		}

		// Add launch date if present
		if product.LaunchDate != nil {
			response["product"].(blaze.Map)["launch_date"] = product.LaunchDate
		}

		// Add image info
		if len(product.Images) > 0 {
			var images []blaze.Map
			for _, img := range product.Images {
				images = append(images, blaze.Map{
					"filename":     img.Filename,
					"size":         img.Size,
					"content_type": img.ContentType,
				})
			}
			response["product"].(blaze.Map)["images"] = images
		}

		return c.JSON(response)
	})

	// Blog Post endpoint
	app.POST("/blog-post", func(c *blaze.Context) error {
		var post BlogPost

		if err := c.BindMultipartForm(&post); err != nil {
			return c.Status(400).JSON(blaze.Map{
				"error":   "Form binding failed",
				"details": err.Error(),
			})
		}

		// Process categories
		if categoriesStr := c.FormValue("categories"); categoriesStr != "" {
			post.Categories = strings.Split(categoriesStr, ",")
			for i := range post.Categories {
				post.Categories[i] = strings.TrimSpace(post.Categories[i])
			}
		}

		response := blaze.Map{
			"message": "Blog post created successfully",
			"post": blaze.Map{
				"title":        post.Title,
				"content":      post.Content,
				"author":       post.Author,
				"published":    post.Published,
				"publish_date": post.PublishDate,
				"categories":   post.Categories,
				"views":        post.Views,
				"rating":       post.Rating,
			},
		}

		// Add featured image info
		if post.FeaturedImg != nil {
			response["post"].(blaze.Map)["featured_image"] = blaze.Map{
				"filename":     post.FeaturedImg.Filename,
				"size":         post.FeaturedImg.Size,
				"content_type": post.FeaturedImg.ContentType,
			}
		}

		// Add attachments info
		if len(post.Attachments) > 0 {
			var attachments []blaze.Map
			for _, att := range post.Attachments {
				attachments = append(attachments, blaze.Map{
					"filename":     att.Filename,
					"size":         att.Size,
					"content_type": att.ContentType,
				})
			}
			response["post"].(blaze.Map)["attachments"] = attachments
		}

		return c.JSON(response)
	})

	// Test endpoint for URL-encoded forms
	app.POST("/url-encoded", func(c *blaze.Context) error {
		var profile UserProfile

		if err := c.BindForm(&profile); err != nil {
			return c.Status(400).JSON(blaze.Map{
				"error":   "Form binding failed",
				"details": err.Error(),
			})
		}

		return c.JSON(blaze.Map{
			"message": "URL-encoded form bound successfully",
			"profile": profile,
		})
	})

	log.Printf("🚀 Form Binding Server starting on http://localhost:8080")
	log.Printf("🔗 Open http://localhost:8080 in your browser")
	log.Fatal(app.ListenAndServe())
}
