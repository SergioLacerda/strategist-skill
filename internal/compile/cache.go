package compile

import (
	"container/list"
	"os"
	"sync"
	"time"
)

type cacheKey struct {
	path  string
	mtime int64
	size  int64
}

type cacheEntry struct {
	key  cacheKey
	data []byte
	// accessedAt tracks recency for LRU eviction.
	accessedAt time.Time
}

// compiledCache is a process-local LRU cache for decompressed compiled artifacts.
// Invalidation is automatic: a key includes path+mtime+size, so any file change
// produces a cache miss. TTL is intentionally absent — mtime is the only signal.
type compiledCache struct {
	mu      sync.Mutex
	lru     *list.List
	index   map[cacheKey]*list.Element
	maxSize int
}

func newCompiledCache(maxSize int) *compiledCache {
	return &compiledCache{
		lru:     list.New(),
		index:   make(map[cacheKey]*list.Element),
		maxSize: maxSize,
	}
}

// Get returns cached data if path+mtime+size matches, nil on miss.
func (c *compiledCache) Get(path string, info os.FileInfo) []byte {
	key := cacheKey{path: path, mtime: info.ModTime().Unix(), size: info.Size()}
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.index[key]
	if !ok {
		return nil
	}
	entry, ok := elem.Value.(*cacheEntry)
	if !ok {
		return nil
	}
	entry.accessedAt = time.Now()
	c.lru.MoveToFront(elem)
	return entry.data
}

// Set stores decompressed data keyed by path+mtime+size.
// If the key already exists, the entry is refreshed. On capacity overflow the
// least-recently-used entry is evicted.
func (c *compiledCache) Set(path string, info os.FileInfo, data []byte) {
	key := cacheKey{path: path, mtime: info.ModTime().Unix(), size: info.Size()}
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.index[key]; ok {
		entry, ok := elem.Value.(*cacheEntry)
		if !ok {
			return
		}
		entry.data = data
		entry.accessedAt = time.Now()
		c.lru.MoveToFront(elem)
		return
	}
	for c.lru.Len() >= c.maxSize {
		back := c.lru.Back()
		if back == nil {
			break
		}
		c.lru.Remove(back)
		if entry, ok := back.Value.(*cacheEntry); ok {
			delete(c.index, entry.key)
		}
	}
	entry := &cacheEntry{key: key, data: data, accessedAt: time.Now()}
	elem := c.lru.PushFront(entry)
	c.index[key] = elem
}

// Len returns the number of entries currently in the cache.
func (c *compiledCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}
