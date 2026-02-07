package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Cache is a simple in-memory cache with expiration
type Cache struct {
	items map[string]*cacheItem
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// New creates a new cache with the given TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*cacheItem),
		ttl:   ttl,
	}
	
	// Start cleanup goroutine
	go c.cleanup()
	
	return c
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	
	// Check if expired
	if time.Now().After(item.expiration) {
		return nil, false
	}
	
	return item.value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*cacheItem)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return len(c.items)
}

// cleanup removes expired items periodically
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// HashKey creates a cache key from multiple strings
func HashKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FileCache caches file contents
type FileCache struct {
	cache *Cache
}

// NewFileCache creates a new file cache
func NewFileCache(ttl time.Duration) *FileCache {
	return &FileCache{
		cache: New(ttl),
	}
}

// Get retrieves cached file content
func (fc *FileCache) Get(path string) ([]byte, bool) {
	val, ok := fc.cache.Get(path)
	if !ok {
		return nil, false
	}
	
	content, ok := val.([]byte)
	return content, ok
}

// Set caches file content
func (fc *FileCache) Set(path string, content []byte) {
	fc.cache.Set(path, content)
}

// Clear clears the cache
func (fc *FileCache) Clear() {
	fc.cache.Clear()
}
