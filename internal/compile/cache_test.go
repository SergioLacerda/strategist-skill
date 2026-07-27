package compile

import (
	"os"
	"testing"
	"time"
)

// fakeFileInfo satisfies os.FileInfo for cache key construction in tests.
type fakeFileInfo struct {
	modTime time.Time
	size    int64
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() os.FileMode  { return 0o644 }
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func newFakeInfo(mtime int64, size int64) os.FileInfo {
	return fakeFileInfo{modTime: time.Unix(mtime, 0), size: size}
}

func TestCompiledCache_Hit(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(8)
	info := newFakeInfo(1000, 512)
	c.Set("a.gz", info, []byte("data"))
	got := c.Get("a.gz", info)
	if string(got) != "data" {
		t.Fatalf("expected cache hit with 'data', got %q", got)
	}
}

func TestCompiledCache_Miss_UnknownPath(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(8)
	info := newFakeInfo(1000, 512)
	if got := c.Get("missing.gz", info); got != nil {
		t.Fatalf("expected nil on miss, got %q", got)
	}
}

func TestCompiledCache_Invalidate_MtimeChange(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(8)
	c.Set("a.gz", newFakeInfo(1000, 512), []byte("old"))
	// same path, different mtime → cache miss (stale)
	if got := c.Get("a.gz", newFakeInfo(2000, 512)); got != nil {
		t.Fatalf("expected nil after mtime change, got %q", got)
	}
}

func TestCompiledCache_Invalidate_SizeChange(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(8)
	c.Set("a.gz", newFakeInfo(1000, 512), []byte("old"))
	// same path+mtime, different size → cache miss
	if got := c.Get("a.gz", newFakeInfo(1000, 999)); got != nil {
		t.Fatalf("expected nil after size change, got %q", got)
	}
}

func TestCompiledCache_Update_SameKey(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(8)
	info := newFakeInfo(1000, 512)
	c.Set("a.gz", info, []byte("v1"))
	c.Set("a.gz", info, []byte("v2"))
	if got := c.Get("a.gz", info); string(got) != "v2" {
		t.Fatalf("expected updated value 'v2', got %q", got)
	}
	if c.Len() != 1 {
		t.Fatalf("expected Len=1 after updating same key, got %d", c.Len())
	}
}

func TestCompiledCache_Eviction_LRU(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(3)
	info := func(n int) os.FileInfo { return newFakeInfo(int64(n), int64(n)) }

	c.Set("a.gz", info(1), []byte("a"))
	c.Set("b.gz", info(2), []byte("b"))
	c.Set("c.gz", info(3), []byte("c"))
	// "a" is least recently used → evicted when "d" is inserted
	c.Set("d.gz", info(4), []byte("d"))

	if c.Len() != 3 {
		t.Fatalf("expected Len=3 after eviction, got %d", c.Len())
	}
	if got := c.Get("a.gz", info(1)); got != nil {
		t.Fatalf("expected 'a.gz' to be evicted, but got %q", got)
	}
	if got := c.Get("d.gz", info(4)); string(got) != "d" {
		t.Fatalf("expected 'd.gz' to be present, got %q", got)
	}
}

func TestCompiledCache_ZeroMaxSize_EvictOverflowStopsOnEmptyList(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(0)
	info := newFakeInfo(1000, 512)
	// With maxSize 0, evictOverflow tries to evict down to capacity but the list
	// starts empty, so it must break instead of looping or panicking on a nil back element.
	c.Set("a.gz", info, []byte("data"))
	if c.Len() != 1 {
		t.Fatalf("expected Len=1 after Set on empty zero-capacity cache, got %d", c.Len())
	}
	if got := c.Get("a.gz", info); string(got) != "data" {
		t.Fatalf("expected 'data', got %q", got)
	}
}

func TestCompiledCache_Eviction_LRU_AccessRenewsOrder(t *testing.T) {
	t.Parallel()
	c := newCompiledCache(3)
	info := func(n int) os.FileInfo { return newFakeInfo(int64(n), int64(n)) }

	c.Set("a.gz", info(1), []byte("a"))
	c.Set("b.gz", info(2), []byte("b"))
	c.Set("c.gz", info(3), []byte("c"))
	// Access "a" so it moves to front — "b" becomes LRU
	_ = c.Get("a.gz", info(1))
	c.Set("d.gz", info(4), []byte("d"))

	if got := c.Get("b.gz", info(2)); got != nil {
		t.Fatalf("expected 'b.gz' to be evicted (was LRU), but got %q", got)
	}
	if got := c.Get("a.gz", info(1)); string(got) != "a" {
		t.Fatalf("expected 'a.gz' to survive (was accessed), got %q", got)
	}
}
