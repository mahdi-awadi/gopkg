// Package lru provides a tiny thread-safe LRU cache with optional TTL.
//
// Built on stdlib container/list. Generic over key (comparable) and
// value (any).
//
// Zero third-party deps.
package lru

import (
	"container/list"
	"sync"
	"time"
)

// Cache is a thread-safe LRU cache with optional per-entry TTL.
//
// Zero value is not usable; construct via New.
type Cache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	ll       *list.List
	items    map[K]*list.Element
	now      func() time.Time
}

type entry[K comparable, V any] struct {
	key      K
	value    V
	expireAt time.Time // zero → no expiry
}

// New returns a cache that evicts least-recently-used entries once size
// exceeds capacity. ttl of 0 means no expiry (pure LRU).
func New[K comparable, V any](capacity int, ttl time.Duration) *Cache[K, V] {
	if capacity < 1 {
		capacity = 1
	}
	return &Cache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[K]*list.Element, capacity),
		now:      time.Now,
	}
}

// Set puts key=value in the cache. If an entry exists it is replaced and
// its recency reset. If the cache is at capacity, the LRU entry is evicted.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if el, ok := c.items[key]; ok {
		en := el.Value.(*entry[K, V])
		en.value = value
		if c.ttl > 0 {
			en.expireAt = now.Add(c.ttl)
		}
		c.ll.MoveToFront(el)
		return
	}

	en := &entry[K, V]{key: key, value: value}
	if c.ttl > 0 {
		en.expireAt = now.Add(c.ttl)
	}
	el := c.ll.PushFront(en)
	c.items[key] = el

	if c.ll.Len() > c.capacity {
		c.removeOldest()
	}
}

// Get returns the value for key and reports whether it was present and
// non-expired. Promotes to most-recent on hit.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	en := el.Value.(*entry[K, V])
	if c.ttl > 0 && !en.expireAt.IsZero() && c.now().After(en.expireAt) {
		c.remove(el)
		return zero, false
	}
	c.ll.MoveToFront(el)
	return en.value, true
}

// Delete removes the key if present.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.remove(el)
	}
}

// Len returns the current number of entries (including possibly-expired).
func (c *Cache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Clear removes all entries.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[K]*list.Element, c.capacity)
}

func (c *Cache[K, V]) removeOldest() {
	if el := c.ll.Back(); el != nil {
		c.remove(el)
	}
}

func (c *Cache[K, V]) remove(el *list.Element) {
	en := el.Value.(*entry[K, V])
	delete(c.items, en.key)
	c.ll.Remove(el)
}
