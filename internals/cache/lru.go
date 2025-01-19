package cache

import (
	"container/list"
	"sync"
)

// defining cacheEntry and Cache struct
type cacheEntry struct {
	key  string
	data []byte
}
type Cache struct {
	mu      sync.Mutex
	maxSize int
	ll      *list.List // DLL for LRU order
	cache   map[string]*list.Element
}

func NewCache(max int) *Cache {
	return &Cache{
		maxSize: max,
		ll:      list.New(),
		cache:   make(map[string]*list.Element),
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem) // update usage
		entry := elem.Value.(*cacheEntry)
		return entry.data, true
	}
	return nil, false
}

func (c *Cache) Add(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key already exists, update data and move to front
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.data = data
		return
	}

	// Add new entry to front
	entry := &cacheEntry{key: key, data: data}
	elem := c.ll.PushFront(entry)
	c.cache[key] = elem

	// Evict least recently used if over capacity
	if c.ll.Len() > c.maxSize {
		c.evict()
	}
}

func (c *Cache) evict() {
	// Remove from back of list (LRU)
	elem := c.ll.Back()
	if elem == nil {
		return
	}
	entry := elem.Value.(*cacheEntry)
	delete(c.cache, entry.key)
	c.ll.Remove(elem)
}
