package cache

import (
	"sync"
	"time"
)

type LruCache[K comparable, V any] struct {
	entries   map[K]V
	lastTime  map[K]time.Time
	itemLimit uint32 // maximum number of items in the cache
	mu        sync.Mutex
}

func NewLruCache[K comparable, V any](itemLimit uint32) *LruCache[K, V] {
	return &LruCache[K, V]{
		entries:   make(map[K]V),
		lastTime:  make(map[K]time.Time),
		itemLimit: itemLimit,
	}
}

func (c *LruCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if ok {
		c.lastTime[key] = time.Now()
	}
	return entry, ok
}

// Set adds a value to the cache with an optional lifetime.
// If adding this item would exceed the item limit, the least recently used item is evicted.
// If lifetime is greater than 0, the item is deleted after the lifetime.
// Otherwise, if lifetime is 0, the item is not going to be expired.
func (c *LruCache[K, V]) Set(key K, value V, lifetime time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.itemLimit > 0 {
		_, exists := c.entries[key]
		if !exists && len(c.entries) >= int(c.itemLimit) {
			var oldestK K
			var oldestT time.Time
			first := true
			for k, t := range c.lastTime {
				if first || t.Before(oldestT) {
					oldestK, oldestT = k, t
					first = false
				}
			}
			if !first {
				delete(c.entries, oldestK)
				delete(c.lastTime, oldestK)
			} else {
				for k := range c.entries { // fallback
					delete(c.entries, k)
					delete(c.lastTime, k)
					break
				}
			}
		}
	}

	c.entries[key] = value
	c.lastTime[key] = time.Now()
	if lifetime > 0 {
		go func() {
			time.Sleep(lifetime)
			c.Delete(key)
		}()
	}
}

func (c *LruCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	delete(c.lastTime, key)
}
