package blaze

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	// Response data
	StatusCode int
	Headers    map[string]string
	Body       []byte

	// Cache metadata
	CreatedAt   time.Time
	ExpiresAt   time.Time
	AccessedAt  int64 // atomic counter
	AccessCount int64 // atomic counter

	// Cache validation
	ETag         string
	LastModified time.Time

	// Size for memory management
	Size int64
}

// CacheStore defines the interface for cache storage
type CacheStore interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry, ttl time.Duration) bool
	Delete(key string) bool
	Clear() int
	Size() int64
	Keys() []string
	Stats() CacheStats
	Cleanup() int
}

// MemoryStore implements CacheStore for in-memory caching
type MemoryStore struct {
	entries    map[string]*CacheEntry
	mu         sync.RWMutex
	maxSize    int64
	maxEntries int
	algorithm  EvictionAlgorithm
	stats      CacheStats
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Sets      int64   `json:"sets"`
	Deletes   int64   `json:"deletes"`
	Evictions int64   `json:"evictions"`
	Size      int64   `json:"size"`
	Entries   int64   `json:"entries"`
	HitRatio  float64 `json:"hit_ratio"`
}

// EvictionAlgorithm defines cache eviction strategies
type EvictionAlgorithm int

const (
	LRU    EvictionAlgorithm = iota // Least Recently Used
	LFU                             // Least Frequently Used
	FIFO                            // First In First Out
	Random                          // Random eviction
)

// CacheOptions configures the cache middleware
type CacheOptions struct {
	// Storage configuration
	Store      CacheStore
	DefaultTTL time.Duration
	MaxAge     time.Duration

	// Memory limits
	MaxSize    int64 // Maximum total cache size in bytes
	MaxEntries int   // Maximum number of cache entries

	// Eviction strategy
	Algorithm EvictionAlgorithm

	// Cache control
	Skipper      func(c *Context) bool
	KeyGenerator func(c *Context) string
	ShouldCache  func(c *Context) bool
	VaryHeaders  []string

	// HTTP cache headers
	Public          bool
	Private         bool
	NoCache         bool
	NoStore         bool
	MustRevalidate  bool
	ProxyRevalidate bool
	Immutable       bool

	// Compression
	EnableCompression bool
	CompressionLevel  int

	// Background tasks
	CleanupInterval         time.Duration
	EnableBackgroundCleanup bool

	// Cache warming
	WarmupURLs []string

	// Debugging
	EnableHeaders bool
	HeaderPrefix  string
}

// DefaultCacheOptions returns sensible defaults for cache configuration
func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		Store:                   NewMemoryStore(100*1024*1024, 10000, LRU), // 100MB, 10k entries
		DefaultTTL:              5 * time.Minute,
		MaxAge:                  1 * time.Hour,
		MaxSize:                 100 * 1024 * 1024, // 100MB
		MaxEntries:              10000,
		Algorithm:               LRU,
		VaryHeaders:             []string{"Accept-Encoding", "Accept"},
		Public:                  true,
		EnableCompression:       true,
		CompressionLevel:        6,
		CleanupInterval:         5 * time.Minute,
		EnableBackgroundCleanup: true,
		EnableHeaders:           true,
		HeaderPrefix:            "X-Cache-",
	}
}

// ProductionCacheOptions returns production-ready cache configuration
func ProductionCacheOptions() *CacheOptions {
	opts := DefaultCacheOptions()
	opts.MaxSize = 512 * 1024 * 1024 // 512MB
	opts.MaxEntries = 50000
	opts.DefaultTTL = 15 * time.Minute
	opts.MaxAge = 24 * time.Hour
	opts.CleanupInterval = 2 * time.Minute
	opts.EnableCompression = true
	opts.CompressionLevel = 9
	opts.MustRevalidate = true
	opts.EnableHeaders = false // Disable debug headers in production
	return opts
}

// NewMemoryStore creates a new in-memory cache store
func NewMemoryStore(maxSize int64, maxEntries int, algorithm EvictionAlgorithm) *MemoryStore {
	store := &MemoryStore{
		entries:    make(map[string]*CacheEntry),
		maxSize:    maxSize,
		maxEntries: maxEntries,
		algorithm:  algorithm,
	}
	return store
}

// Cache creates a new cache middleware
func Cache(opts *CacheOptions) MiddlewareFunc {
	if opts == nil {
		opts = DefaultCacheOptions()
	}

	// Validate options
	if err := validateCacheOptions(opts); err != nil {
		panic(fmt.Sprintf("Cache middleware configuration error: %v", err))
	}

	// Initialize store if not provided
	if opts.Store == nil {
		opts.Store = NewMemoryStore(opts.MaxSize, opts.MaxEntries, opts.Algorithm)
	}

	// Start background cleanup
	if opts.EnableBackgroundCleanup {
		go startBackgroundCleanup(opts.Store, opts.CleanupInterval)
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Skip cache if skipper returns true
			if opts.Skipper != nil && opts.Skipper(c) {
				return next(c)
			}

			// Only cache GET and HEAD requests
			if !isCacheableMethod(c.Method()) {
				return next(c)
			}

			// Generate cache key
			cacheKey := generateCacheKey(c, opts)

			// Try to get from cache
			if entry, hit := opts.Store.Get(cacheKey); hit && !isExpired(entry) {
				// Validate cache with conditional headers
				if isNotModified(c, entry) {
					return serveCachedResponse(c, entry, opts, true)
				}
				return serveCachedResponse(c, entry, opts, false)
			}

			// Cache miss - execute handler and cache response
			return cacheResponse(c, next, cacheKey, opts)
		}
	}
}

// CacheResponse caches responses for specific routes
func CacheResponse(ttl time.Duration, opts ...*CacheOptions) MiddlewareFunc {
	var cacheOpts *CacheOptions
	if len(opts) > 0 {
		cacheOpts = opts[0]
	} else {
		cacheOpts = DefaultCacheOptions()
	}
	cacheOpts.DefaultTTL = ttl
	return Cache(cacheOpts)
}

// CacheStatic caches static files with longer TTL
func CacheStatic(opts ...*CacheOptions) MiddlewareFunc {
	var cacheOpts *CacheOptions
	if len(opts) > 0 {
		cacheOpts = opts[0]
	} else {
		cacheOpts = DefaultCacheOptions()
	}

	// Configure for static files
	cacheOpts.DefaultTTL = 24 * time.Hour
	cacheOpts.MaxAge = 7 * 24 * time.Hour // 1 week
	cacheOpts.Public = true
	cacheOpts.Immutable = true

	cacheOpts.ShouldCache = func(c *Context) bool {
		path := c.Path()
		staticExts := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".ico", ".svg", ".woff", ".woff2", ".ttf"}
		for _, ext := range staticExts {
			if strings.HasSuffix(path, ext) {
				return true
			}
		}
		return false
	}

	return Cache(cacheOpts)
}

// CacheAPI caches API responses with shorter TTL
func CacheAPI(ttl time.Duration) MiddlewareFunc {
	opts := DefaultCacheOptions()
	opts.DefaultTTL = ttl
	opts.Private = true
	opts.VaryHeaders = []string{"Authorization", "Accept", "Accept-Encoding"}

	opts.ShouldCache = func(c *Context) bool {
		// Only cache successful responses
		statusCode := c.Response().StatusCode()
		return statusCode >= 200 && statusCode < 300
	}

	return Cache(opts)
}

// validateCacheOptions validates cache configuration
func validateCacheOptions(opts *CacheOptions) error {
	if opts.DefaultTTL <= 0 {
		return fmt.Errorf("default TTL must be positive")
	}

	if opts.MaxSize <= 0 {
		return fmt.Errorf("max size must be positive")
	}

	if opts.MaxEntries <= 0 {
		return fmt.Errorf("max entries must be positive")
	}

	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = 5 * time.Minute
	}

	return nil
}

// generateCacheKey creates a unique cache key for the request
func generateCacheKey(c *Context, opts *CacheOptions) string {
	if opts.KeyGenerator != nil {
		return opts.KeyGenerator(c)
	}

	// Base key components
	method := c.Method()
	path := c.Path()
	query := string(c.RequestCtx.QueryArgs().QueryString())

	// Add vary headers to key
	var varyParts []string
	for _, header := range opts.VaryHeaders {
		value := c.Header(header)
		if value != "" {
			varyParts = append(varyParts, header+":"+value)
		}
	}

	// Create composite key
	keyComponents := []string{method, path}
	if query != "" {
		keyComponents = append(keyComponents, query)
	}
	if len(varyParts) > 0 {
		keyComponents = append(keyComponents, strings.Join(varyParts, "|"))
	}

	fullKey := strings.Join(keyComponents, "|")

	// Hash for consistent key length
	hash := md5.Sum([]byte(fullKey))
	return hex.EncodeToString(hash[:])
}

// cacheResponse executes the handler and caches the response
func cacheResponse(c *Context, next HandlerFunc, cacheKey string, opts *CacheOptions) error {
	// Execute handler
	err := next(c)
	if err != nil {
		return err
	}

	// Check if response should be cached
	if !shouldCacheResponse(c, opts) {
		return nil
	}

	// Create cache entry
	entry := &CacheEntry{
		StatusCode:   c.Response().StatusCode(),
		Headers:      make(map[string]string),
		Body:         make([]byte, len(c.Response().Body())),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(opts.DefaultTTL),
		ETag:         generateETag(c.Response().Body()),
		LastModified: time.Now(),
		Size:         int64(len(c.Response().Body())),
	}

	// Copy response body
	copy(entry.Body, c.Response().Body())

	// Copy cacheable headers
	c.Response().Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if isCacheableHeader(keyStr) {
			entry.Headers[keyStr] = string(value)
		}
	})

	// Set cache control headers
	setCacheControlHeaders(c, opts)

	// Add debug headers
	if opts.EnableHeaders {
		c.SetHeader(opts.HeaderPrefix+"Status", "MISS")
		c.SetHeader(opts.HeaderPrefix+"Key", cacheKey[:16]) // First 16 chars of key
		c.SetHeader(opts.HeaderPrefix+"TTL", strconv.Itoa(int(opts.DefaultTTL.Seconds())))
	}

	// Store in cache
	opts.Store.Set(cacheKey, entry, opts.DefaultTTL)

	return nil
}

// serveCachedResponse serves a response from cache
func serveCachedResponse(c *Context, entry *CacheEntry, opts *CacheOptions, notModified bool) error {
	// Update access statistics
	atomic.AddInt64(&entry.AccessedAt, time.Now().Unix())
	atomic.AddInt64(&entry.AccessCount, 1)

	if notModified {
		// Send 304 Not Modified
		c.Status(fasthttp.StatusNotModified)
		setConditionalHeaders(c, entry)

		if opts.EnableHeaders {
			c.SetHeader(opts.HeaderPrefix+"Status", "HIT-304")
		}
		return nil
	}

	// Set status code
	c.Status(entry.StatusCode)

	// Set cached headers
	for key, value := range entry.Headers {
		c.SetHeader(key, value)
	}

	// Set conditional headers
	setConditionalHeaders(c, entry)

	// Set cache control headers
	setCacheControlHeaders(c, opts)

	// Add debug headers
	if opts.EnableHeaders {
		c.SetHeader(opts.HeaderPrefix+"Status", "HIT")
		c.SetHeader(opts.HeaderPrefix+"Age", strconv.Itoa(int(time.Since(entry.CreatedAt).Seconds())))
		c.SetHeader(opts.HeaderPrefix+"Expires", entry.ExpiresAt.Format(time.RFC1123))
	}

	// Write cached body
	c.Response().SetBody(entry.Body)

	return nil
}

// shouldCacheResponse determines if a response should be cached
func shouldCacheResponse(c *Context, opts *CacheOptions) bool {
	// Check custom should cache function
	if opts.ShouldCache != nil {
		return opts.ShouldCache(c)
	}

	// Only cache successful responses
	statusCode := c.Response().StatusCode()
	if statusCode < 200 || statusCode >= 300 {
		return false
	}

	// Don't cache responses with cache-control: no-cache or no-store
	cacheControl := c.Header("Cache-Control")
	if strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store") {
		return false
	}

	return true
}

// isCacheableMethod checks if HTTP method is cacheable
func isCacheableMethod(method string) bool {
	return method == "GET" || method == "HEAD"
}

// isCacheableHeader checks if header should be cached
func isCacheableHeader(header string) bool {
	nonCacheableHeaders := []string{
		"Set-Cookie",
		"Authorization",
		"Proxy-Authorization",
		"Cache-Control",
		"Expires",
		"Date",
		"Age",
		"Vary",
	}

	header = strings.ToLower(header)
	for _, nonCacheable := range nonCacheableHeaders {
		if strings.ToLower(nonCacheable) == header {
			return false
		}
	}
	return true
}

// isExpired checks if cache entry is expired
func isExpired(entry *CacheEntry) bool {
	return time.Now().After(entry.ExpiresAt)
}

// isNotModified checks conditional headers for 304 responses
func isNotModified(c *Context, entry *CacheEntry) bool {
	// Check If-None-Match (ETag)
	if etag := c.Header("If-None-Match"); etag != "" {
		return etag == entry.ETag || etag == "*"
	}

	// Check If-Modified-Since
	if modSince := c.Header("If-Modified-Since"); modSince != "" {
		if parsedTime, err := time.Parse(time.RFC1123, modSince); err == nil {
			return !entry.LastModified.After(parsedTime)
		}
	}

	return false
}

// generateETag generates an ETag for response body
func generateETag(body []byte) string {
	hash := md5.Sum(body)
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// setConditionalHeaders sets ETag and Last-Modified headers
func setConditionalHeaders(c *Context, entry *CacheEntry) {
	if entry.ETag != "" {
		c.SetHeader("ETag", entry.ETag)
	}

	if !entry.LastModified.IsZero() {
		c.SetHeader("Last-Modified", entry.LastModified.Format(time.RFC1123))
	}
}

// setCacheControlHeaders sets cache control headers
func setCacheControlHeaders(c *Context, opts *CacheOptions) {
	var directives []string

	if opts.Public {
		directives = append(directives, "public")
	}
	if opts.Private {
		directives = append(directives, "private")
	}
	if opts.NoCache {
		directives = append(directives, "no-cache")
	}
	if opts.NoStore {
		directives = append(directives, "no-store")
	}
	if opts.MustRevalidate {
		directives = append(directives, "must-revalidate")
	}
	if opts.ProxyRevalidate {
		directives = append(directives, "proxy-revalidate")
	}
	if opts.Immutable {
		directives = append(directives, "immutable")
	}

	// Add max-age
	if opts.MaxAge > 0 {
		directives = append(directives, fmt.Sprintf("max-age=%d", int(opts.MaxAge.Seconds())))
	}

	if len(directives) > 0 {
		c.SetHeader("Cache-Control", strings.Join(directives, ", "))
	}

	// Set Vary header
	if len(opts.VaryHeaders) > 0 {
		c.SetHeader("Vary", strings.Join(opts.VaryHeaders, ", "))
	}
}

// MemoryStore implementation methods
func (ms *MemoryStore) Get(key string) (*CacheEntry, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entry, exists := ms.entries[key]
	if !exists || isExpired(entry) {
		if exists {
			// Clean up expired entry
			delete(ms.entries, key)
			atomic.AddInt64(&ms.stats.Size, -entry.Size)
			atomic.AddInt64(&ms.stats.Entries, -1)
		}
		atomic.AddInt64(&ms.stats.Misses, 1)
		return nil, false
	}

	atomic.AddInt64(&ms.stats.Hits, 1)
	return entry, true
}

func (ms *MemoryStore) Set(key string, entry *CacheEntry, ttl time.Duration) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if we need to evict
	if ms.needsEviction(entry.Size) {
		ms.evictEntries(entry.Size)
	}

	// Remove existing entry if present
	if existing, exists := ms.entries[key]; exists {
		atomic.AddInt64(&ms.stats.Size, -existing.Size)
		atomic.AddInt64(&ms.stats.Entries, -1)
	}

	// Set expiration if TTL provided
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	ms.entries[key] = entry
	atomic.AddInt64(&ms.stats.Size, entry.Size)
	atomic.AddInt64(&ms.stats.Entries, 1)
	atomic.AddInt64(&ms.stats.Sets, 1)

	return true
}

func (ms *MemoryStore) Delete(key string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if entry, exists := ms.entries[key]; exists {
		delete(ms.entries, key)
		atomic.AddInt64(&ms.stats.Size, -entry.Size)
		atomic.AddInt64(&ms.stats.Entries, -1)
		atomic.AddInt64(&ms.stats.Deletes, 1)
		return true
	}
	return false
}

func (ms *MemoryStore) Clear() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	count := len(ms.entries)
	ms.entries = make(map[string]*CacheEntry)
	atomic.StoreInt64(&ms.stats.Size, 0)
	atomic.StoreInt64(&ms.stats.Entries, 0)

	return count
}

func (ms *MemoryStore) Size() int64 {
	return atomic.LoadInt64(&ms.stats.Size)
}

func (ms *MemoryStore) Keys() []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	keys := make([]string, 0, len(ms.entries))
	for key := range ms.entries {
		keys = append(keys, key)
	}
	return keys
}

func (ms *MemoryStore) Stats() CacheStats {
	stats := ms.stats

	// Calculate hit ratio
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}

	return stats
}

func (ms *MemoryStore) Cleanup() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var removed int
	now := time.Now()

	for key, entry := range ms.entries {
		if now.After(entry.ExpiresAt) {
			delete(ms.entries, key)
			atomic.AddInt64(&ms.stats.Size, -entry.Size)
			atomic.AddInt64(&ms.stats.Entries, -1)
			removed++
		}
	}

	return removed
}

// needsEviction checks if eviction is needed
func (ms *MemoryStore) needsEviction(newEntrySize int64) bool {
	if len(ms.entries) >= ms.maxEntries {
		return true
	}

	if ms.stats.Size+newEntrySize > ms.maxSize {
		return true
	}

	return false
}

// evictEntries removes entries based on eviction algorithm
func (ms *MemoryStore) evictEntries(sizeNeeded int64) {
	var freedSize int64

	switch ms.algorithm {
	case LRU:
		ms.evictLRU(&freedSize, sizeNeeded)
	case LFU:
		ms.evictLFU(&freedSize, sizeNeeded)
	case FIFO:
		ms.evictFIFO(&freedSize, sizeNeeded)
	case Random:
		ms.evictRandom(&freedSize, sizeNeeded)
	}
}

// evictLRU removes least recently used entries
func (ms *MemoryStore) evictLRU(freedSize *int64, sizeNeeded int64) {
	type keyEntry struct {
		key        string
		accessedAt int64
	}

	var entries []keyEntry
	for key, entry := range ms.entries {
		entries = append(entries, keyEntry{
			key:        key,
			accessedAt: atomic.LoadInt64(&entry.AccessedAt),
		})
	}

	// Sort by access time (oldest first)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].accessedAt > entries[j].accessedAt {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest entries
	for _, entry := range entries {
		if *freedSize >= sizeNeeded && len(ms.entries) < ms.maxEntries {
			break
		}

		if cached := ms.entries[entry.key]; cached != nil {
			*freedSize += cached.Size
			delete(ms.entries, entry.key)
			atomic.AddInt64(&ms.stats.Size, -cached.Size)
			atomic.AddInt64(&ms.stats.Entries, -1)
			atomic.AddInt64(&ms.stats.Evictions, 1)
		}
	}
}

// evictLFU removes least frequently used entries
func (ms *MemoryStore) evictLFU(freedSize *int64, sizeNeeded int64) {
	type keyEntry struct {
		key         string
		accessCount int64
	}

	var entries []keyEntry
	for key, entry := range ms.entries {
		entries = append(entries, keyEntry{
			key:         key,
			accessCount: atomic.LoadInt64(&entry.AccessCount),
		})
	}

	// Sort by access count (least first)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].accessCount > entries[j].accessCount {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove least used entries
	for _, entry := range entries {
		if *freedSize >= sizeNeeded && len(ms.entries) < ms.maxEntries {
			break
		}

		if cached := ms.entries[entry.key]; cached != nil {
			*freedSize += cached.Size
			delete(ms.entries, entry.key)
			atomic.AddInt64(&ms.stats.Size, -cached.Size)
			atomic.AddInt64(&ms.stats.Entries, -1)
			atomic.AddInt64(&ms.stats.Evictions, 1)
		}
	}
}

// evictFIFO removes first in first out entries
func (ms *MemoryStore) evictFIFO(freedSize *int64, sizeNeeded int64) {
	type keyEntry struct {
		key       string
		createdAt time.Time
	}

	var entries []keyEntry
	for key, entry := range ms.entries {
		entries = append(entries, keyEntry{
			key:       key,
			createdAt: entry.CreatedAt,
		})
	}

	// Sort by creation time (oldest first)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].createdAt.After(entries[j].createdAt) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest entries
	for _, entry := range entries {
		if *freedSize >= sizeNeeded && len(ms.entries) < ms.maxEntries {
			break
		}

		if cached := ms.entries[entry.key]; cached != nil {
			*freedSize += cached.Size
			delete(ms.entries, entry.key)
			atomic.AddInt64(&ms.stats.Size, -cached.Size)
			atomic.AddInt64(&ms.stats.Entries, -1)
			atomic.AddInt64(&ms.stats.Evictions, 1)
		}
	}
}

// evictRandom removes random entries
func (ms *MemoryStore) evictRandom(freedSize *int64, sizeNeeded int64) {
	for key, entry := range ms.entries {
		if *freedSize >= sizeNeeded && len(ms.entries) < ms.maxEntries {
			break
		}

		*freedSize += entry.Size
		delete(ms.entries, key)
		atomic.AddInt64(&ms.stats.Size, -entry.Size)
		atomic.AddInt64(&ms.stats.Entries, -1)
		atomic.AddInt64(&ms.stats.Evictions, 1)
	}
}

// startBackgroundCleanup starts the cleanup routine
func startBackgroundCleanup(store CacheStore, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		store.Cleanup()
	}
}

// CacheStatus returns cache statistics
func CacheStatus(c *Context) error {
	// This would need access to the cache store
	// Implementation depends on how you want to expose cache status
	return c.JSON(Map{
		"message": "Cache status endpoint - implement based on your needs",
	})
}

// InvalidateCache provides cache invalidation functionality
func InvalidateCache(store CacheStore, pattern string) int {
	keys := store.Keys()
	var removed int

	for _, key := range keys {
		if strings.Contains(key, pattern) {
			if store.Delete(key) {
				removed++
			}
		}
	}

	return removed
}

// CacheClearAll clears all cache entries
func CacheClearAll(store CacheStore) int {
	return store.Clear()
}
