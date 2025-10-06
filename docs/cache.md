# Cache Documentation

Complete guide to HTTP response caching in the Blaze web framework for improved performance and reduced server load.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Basic Usage](#basic-usage)
- [Cache Strategies](#cache-strategies)
- [Cache Control](#cache-control)
- [Cache Storage](#cache-storage)
- [Eviction Policies](#eviction-policies)
- [Cache Invalidation](#cache-invalidation)
- [Performance Tuning](#performance-tuning)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Blaze provides a powerful, flexible caching middleware that can:

- **Cache HTTP responses** in memory
- **Reduce server load** by serving cached responses
- **Improve performance** with sub-millisecond response times
- **Support multiple eviction strategies** (LRU, LFU, FIFO, Random)
- **Handle cache invalidation** with patterns
- **Implement HTTP caching standards** (ETags, Last-Modified, Cache-Control)
- **Provide cache statistics** for monitoring

### Cache Benefits

- **Performance**: Serve cached responses in microseconds
- **Scalability**: Handle more requests with same resources
- **Cost**: Reduce database queries and external API calls
- **Reliability**: Serve stale content during outages

## Configuration

### CacheOptions

```go
type CacheOptions struct {
    // Storage configuration
    Store             CacheStore        // Cache storage backend
    DefaultTTL        time.Duration     // Default time-to-live
    MaxAge            time.Duration     // Maximum cache age
    
    // Memory limits
    MaxSize           int64             // Maximum total cache size (bytes)
    MaxEntries        int               // Maximum number of entries
    
    // Eviction strategy
    Algorithm         EvictionAlgorithm // LRU, LFU, FIFO, Random
    
    // Cache control
    Skipper           func(c *Context) bool
    KeyGenerator      func(c *Context) string
    ShouldCache       func(c *Context) bool
    VaryHeaders       []string          // Headers that affect cache key
    
    // HTTP cache headers
    Public            bool              // Cache-Control: public
    Private           bool              // Cache-Control: private
    NoCache           bool              // Cache-Control: no-cache
    NoStore           bool              // Cache-Control: no-store
    MustRevalidate    bool              // Cache-Control: must-revalidate
    ProxyRevalidate   bool              // Cache-Control: proxy-revalidate
    Immutable         bool              // Cache-Control: immutable
    
    // Compression
    EnableCompression bool              // Compress cached responses
    CompressionLevel  int               // 0-9 (6 is default)
    
    // Background tasks
    CleanupInterval        time.Duration // Cleanup frequency
    EnableBackgroundCleanup bool         // Auto cleanup expired entries
    
    // Cache warming
    WarmupURLs        []string          // URLs to pre-cache on startup
    
    // Debugging
    EnableHeaders     bool              // Add X-Cache-* headers
    HeaderPrefix      string            // Prefix for cache headers
}
```

### Default Configuration

```go
config := blaze.DefaultCacheOptions()
// MaxSize: 100 MB
// MaxEntries: 10,000
// Algorithm: LRU
// DefaultTTL: 5 minutes
// MaxAge: 1 hour
// Public: true
// EnableCompression: true
// CompressionLevel: 6
// CleanupInterval: 5 minutes
// EnableBackgroundCleanup: true
// EnableHeaders: true
```

### Production Configuration

```go
config := blaze.ProductionCacheOptions()
// MaxSize: 512 MB
// MaxEntries: 50,000
// DefaultTTL: 15 minutes
// MaxAge: 24 hours
// CompressionLevel: 9 (maximum)
// MustRevalidate: true
// EnableHeaders: false (no debug headers)
```

## Basic Usage

### Global Caching

Cache all cacheable responses:

```go
app := blaze.New()

// Use default configuration
app.Use(blaze.Cache(nil))

// Or custom configuration
opts := blaze.DefaultCacheOptions()
opts.DefaultTTL = 10 * time.Minute
app.Use(blaze.Cache(opts))
```

### Route-Specific Caching

Cache specific routes with custom TTL:

```go
// Cache for 5 minutes
app.GET("/api/products", handler, 
    blaze.CacheResponse(5 * time.Minute))

// Cache for 1 hour
app.GET("/api/categories", handler,
    blaze.CacheResponse(time.Hour))
```

### Static File Caching

Aggressive caching for static assets:

```go
// Cache static files for 24 hours
app.Use(blaze.CacheStatic())

// Custom static cache configuration
opts := blaze.CacheOptions{
    DefaultTTL: 7 * 24 * time.Hour,  // 1 week
    MaxAge:     30 * 24 * time.Hour, // 30 days
    Public:     true,
    Immutable:  true,
}
app.Use(blaze.CacheStatic(opts))
```

### API Response Caching

Short-lived caching for API responses:

```go
// Cache API responses for 1 minute
app.Use(blaze.CacheAPI(time.Minute))

// Group-specific caching
api := app.Group("/api")
api.Use(blaze.CacheAPI(30 * time.Second))
```

## Cache Strategies

### Cache Everything (Development)

```go
opts := blaze.DefaultCacheOptions()
opts.DefaultTTL = 1 * time.Minute

app.Use(blaze.Cache(opts))
```

### Selective Caching

Cache only specific responses:

```go
opts := blaze.DefaultCacheOptions()
opts.ShouldCache = func(c *blaze.Context) bool {
    // Only cache successful GET requests
    if c.Method() != "GET" {
        return false
    }
    
    statusCode := c.Response().StatusCode()
    return statusCode >= 200 && statusCode < 300
}

app.Use(blaze.Cache(opts))
```

### Skip Caching for Authenticated Requests

```go
opts := blaze.DefaultCacheOptions()
opts.Skipper = func(c *blaze.Context) bool {
    // Skip cache if Authorization header present
    return c.Header("Authorization") != ""
}

app.Use(blaze.Cache(opts))
```

### Vary by Headers

Cache different responses based on request headers:

```go
opts := blaze.DefaultCacheOptions()
opts.VaryHeaders = []string{
    "Accept-Language",  // Different language versions
    "Accept-Encoding",  // Different compression
    "Authorization",    // Per-user caching
}

app.Use(blaze.Cache(opts))
```

## Cache Control

### Cache Control Headers

Set standard HTTP cache control directives:

```go
opts := blaze.CacheOptions{
    // Public cache (CDN, proxies)
    Public: true,
    MaxAge: 1 * time.Hour,
    
    // Or private cache (browser only)
    Private: true,
    
    // Force revalidation
    MustRevalidate: true,
    
    // Never change (immutable)
    Immutable: true,
    
    // No caching
    NoCache: false,
    NoStore: false,
}
```

### Conditional Requests

Automatic support for conditional requests:

```go
// Generates ETag and Last-Modified headers
// Handles If-None-Match and If-Modified-Since
// Returns 304 Not Modified when appropriate

app.GET("/api/data", func(c *blaze.Context) error {
    return c.JSON(data)
})
```

## Cache Storage

### Memory Store

Default in-memory storage:

```go
// Create memory store
store := blaze.NewMemoryStore(
    100 * 1024 * 1024, // 100 MB max size
    10000,              // 10,000 max entries
    blaze.LRU,          // LRU eviction
)

opts := blaze.CacheOptions{
    Store: store,
}

app.Use(blaze.Cache(opts))
```

### Custom Store

Implement `CacheStore` interface:

```go
type CacheStore interface {
    Get(key string) (CacheEntry, bool)
    Set(key string, entry CacheEntry, ttl time.Duration) bool
    Delete(key string) bool
    Clear() int
    Size() int64
    Keys() []string
    Stats() CacheStats
    Cleanup() int
}

// Example: Redis-backed cache store
type RedisStore struct {
    client *redis.Client
}

func (r *RedisStore) Get(key string) (CacheEntry, bool) {
    // Redis implementation
}
```

## Eviction Policies

### LRU (Least Recently Used)

Evicts least recently accessed entries (default):

```go
opts := blaze.CacheOptions{
    Algorithm: blaze.LRU,
}
```

**Best for**: General-purpose caching where recently accessed items are likely to be accessed again.

### LFU (Least Frequently Used)

Evicts least frequently accessed entries:

```go
opts := blaze.CacheOptions{
    Algorithm: blaze.LFU,
}
```

**Best for**: Caching popular content with stable access patterns.

### FIFO (First In First Out)

Evicts oldest entries first:

```go
opts := blaze.CacheOptions{
    Algorithm: blaze.FIFO,
}
```

**Best for**: Time-sensitive data where older entries become stale.

### Random

Evicts random entries:

```go
opts := blaze.CacheOptions{
    Algorithm: blaze.Random,
}
```

**Best for**: Simple, fast eviction when access patterns are unpredictable.

## Cache Invalidation

### Invalidate by Pattern

```go
// Invalidate all product-related cache entries
store := opts.Store
removed := blaze.InvalidateCache(store, "/api/products")
log.Printf("Invalidated %d cache entries", removed)
```

### Clear All Cache

```go
store := opts.Store
count := blaze.CacheClearAll(store)
log.Printf("Cleared %d cache entries", count)
```

### Manual Invalidation

```go
// Delete specific cache entry
store.Delete("GET:/api/products/123")

// Clear entire cache
store.Clear()
```

### Automatic Invalidation

Invalidate on mutations:

```go
app.POST("/api/products", func(c *blaze.Context) error {
    // Create product
    product, err := createProduct(c)
    if err != nil {
        return err
    }
    
    // Invalidate product list cache
    store := c.Locals("cache_store").(blaze.CacheStore)
    blaze.InvalidateCache(store, "/api/products")
    
    return c.Status(201).JSON(product)
})
```

## Performance Tuning

### Memory Management

```go
opts := blaze.CacheOptions{
    MaxSize:    512 * 1024 * 1024, // 512 MB
    MaxEntries: 50000,              // 50k entries
    
    // Aggressive cleanup
    CleanupInterval: 1 * time.Minute,
    EnableBackgroundCleanup: true,
}
```

### Compression

```go
opts := blaze.CacheOptions{
    EnableCompression: true,
    CompressionLevel:  9, // Maximum compression
}

// Trade-off: Higher compression = less memory, more CPU
// Level 6 is balanced for most use cases
```

### Cache Warming

Pre-populate cache on startup:

```go
opts := blaze.CacheOptions{
    WarmupURLs: []string{
        "/api/products",
        "/api/categories",
        "/api/popular",
    },
}

// Warm up cache after server starts
go warmupCache(opts.WarmupURLs)
```

### TTL Strategies

```go
// Short TTL for dynamic data
app.GET("/api/live-feed", handler,
    blaze.CacheResponse(10 * time.Second))

// Medium TTL for semi-static data
app.GET("/api/products", handler,
    blaze.CacheResponse(5 * time.Minute))

// Long TTL for static data
app.GET("/api/categories", handler,
    blaze.CacheResponse(1 * time.Hour))

// Very long TTL for immutable data
app.GET("/api/v1/specs", handler,
    blaze.CacheResponse(24 * time.Hour))
```

## Best Practices

### 1. Use Appropriate TTLs

```go
// ✅ Good - specific TTLs based on data volatility
app.GET("/api/stock-prices", handler,
    blaze.CacheResponse(5 * time.Second))  // Real-time data

app.GET("/api/products", handler,
    blaze.CacheResponse(5 * time.Minute))  // Changes hourly

app.GET("/api/config", handler,
    blaze.CacheResponse(1 * time.Hour))    // Changes rarely

// ❌ Bad - one-size-fits-all TTL
app.Use(blaze.Cache(blaze.DefaultCacheOptions()))
```

### 2. Cache Selectively

```go
// ✅ Good - cache only GET requests
opts.ShouldCache = func(c *blaze.Context) bool {
    return c.Method() == "GET" && c.Response().StatusCode() == 200
}

// ❌ Bad - cache everything including POSTs
app.Use(blaze.Cache(nil))
```

### 3. Use Vary Headers

```go
// ✅ Good - vary by relevant headers
opts.VaryHeaders = []string{"Accept-Language", "Accept-Encoding"}

// ❌ Bad - no vary headers (serves wrong version)
opts.VaryHeaders = nil
```

### 4. Monitor Cache Performance

```go
// Add cache statistics endpoint
app.GET("/admin/cache/stats", func(c *blaze.Context) error {
    store := opts.Store
    stats := store.Stats()
    
    return c.JSON(blaze.Map{
        "hits":        stats.Hits,
        "misses":      stats.Misses,
        "hit_ratio":   stats.HitRatio,
        "size":        stats.Size,
        "entries":     stats.Entries,
        "evictions":   stats.Evictions,
    })
})
```

### 5. Implement Cache Invalidation

```go
// ✅ Good - invalidate on data changes
app.PUT("/api/products/:id", func(c *blaze.Context) error {
    // Update product
    err := updateProduct(c.Param("id"), data)
    if err != nil {
        return err
    }
    
    // Invalidate related caches
    store.Delete("GET:/api/products/" + c.Param("id"))
    blaze.InvalidateCache(store, "/api/products")
    
    return c.JSON(result)
})

// ❌ Bad - never invalidate (serves stale data)
```

## Examples

### Complete Cache Setup

```go
func main() {
    app := blaze.New()
    
    // Create cache store
    store := blaze.NewMemoryStore(
        256 * 1024 * 1024, // 256 MB
        25000,              // 25k entries
        blaze.LRU,
    )
    
    // Configure caching
    opts := blaze.CacheOptions{
        Store:                  store,
        DefaultTTL:             5 * time.Minute,
        MaxAge:                 1 * time.Hour,
        Public:                 true,
        EnableCompression:      true,
        CompressionLevel:       6,
        CleanupInterval:        2 * time.Minute,
        EnableBackgroundCleanup: true,
        EnableHeaders:          true,
        VaryHeaders:            []string{"Accept-Encoding"},
        
        // Only cache successful GET requests
        ShouldCache: func(c *blaze.Context) bool {
            if c.Method() != "GET" {
                return false
            }
            status := c.Response().StatusCode()
            return status >= 200 && status < 300
        },
        
        // Skip cache for authenticated requests
        Skipper: func(c *blaze.Context) bool {
            return c.Header("Authorization") != ""
        },
    }
    
    // Apply cache middleware
    app.Use(blaze.Cache(opts))
    
    // Routes
    app.GET("/api/products", getProducts)
    app.GET("/api/products/:id", getProduct)
    
    // Cache management endpoints
    admin := app.Group("/admin")
    admin.GET("/cache/stats", getCacheStats(store))
    admin.POST("/cache/clear", clearCache(store))
    
    app.Listen(":8080")
}
```

### Cache with Different TTLs per Route

```go
// Static content - cache for 1 week
app.Static("/static", "./public",
    blaze.CacheStatic())

// API endpoints - different TTLs
api := app.Group("/api")

// Frequently changing data - 30 seconds
api.GET("/live-feed", handler,
    blaze.CacheResponse(30 * time.Second))

// Moderately changing data - 5 minutes
api.GET("/products", handler,
    blaze.CacheResponse(5 * time.Minute))

// Rarely changing data - 1 hour
api.GET("/categories", handler,
    blaze.CacheResponse(1 * time.Hour))

// Immutable data - 24 hours
api.GET("/v1/schema", handler,
    blaze.CacheResponse(24 * time.Hour))
```

### Cache Invalidation Pattern

```go
func setupRoutes(app *blaze.App, store blaze.CacheStore) {
    // GET - cacheable
    app.GET("/api/products", func(c *blaze.Context) error {
        products, err := db.GetProducts()
        if err != nil {
            return err
        }
        return c.JSON(products)
    }, blaze.CacheResponse(5 * time.Minute))
    
    // POST - invalidate cache
    app.POST("/api/products", func(c *blaze.Context) error {
        var req CreateProductRequest
        if err := c.BindJSON(&req); err != nil {
            return err
        }
        
        product, err := db.CreateProduct(req)
        if err != nil {
            return err
        }
        
        // Invalidate product list cache
        blaze.InvalidateCache(store, "/api/products")
        
        return c.Status(201).JSON(product)
    })
    
    // PUT - invalidate specific and list cache
    app.PUT("/api/products/:id", func(c *blaze.Context) error {
        id := c.Param("id")
        
        var req UpdateProductRequest
        if err := c.BindJSON(&req); err != nil {
            return err
        }
        
        product, err := db.UpdateProduct(id, req)
        if err != nil {
            return err
        }
        
        // Invalidate specific product and list
        store.Delete("GET:/api/products/" + id)
        blaze.InvalidateCache(store, "/api/products")
        
        return c.JSON(product)
    })
}
```

---

For more information:
- [Middleware Guide](./middleware.md)
- [Configuration Guide](./configuration.md)