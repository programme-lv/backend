package cache_test

import (
	"testing"
	"time"

	"github.com/programme-lv/backend/common/cache"
)

func TestLRU_BasicSetGet(t *testing.T) {
	c := cache.NewLruCache[string, int](10)
	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected miss before set")
	}
	c.Set("a", 1, 0)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("expected hit=1, got ok=%v v=%v", ok, v)
	}
}

func TestLRU_EvictionRespectsRecentAccess(t *testing.T) {
	c := cache.NewLruCache[string, int](2)
	c.Set("a", 1, 0)
	time.Sleep(1 * time.Millisecond)
	c.Set("b", 2, 0)
	// make "a" most recently used
	time.Sleep(1 * time.Millisecond)
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected to find a")
	}
	// adding "c" should evict least-recently-used: "b"
	time.Sleep(1 * time.Millisecond)
	c.Set("c", 3, 0)

	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected b to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatalf("expected a to remain")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatalf("expected c to be present")
	}
}

func TestLRU_Expiry(t *testing.T) {
	c := cache.NewLruCache[string, int](10)
	c.Set("x", 42, 20*time.Millisecond)
	if _, ok := c.Get("x"); !ok {
		t.Fatalf("expected immediate presence")
	}
	time.Sleep(35 * time.Millisecond)
	if _, ok := c.Get("x"); ok {
		t.Fatalf("expected expired entry to be gone")
	}
}

func TestLRU_Delete(t *testing.T) {
	c := cache.NewLruCache[string, int](10)
	c.Set("k", 7, 0)
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Fatalf("expected deleted entry to be gone")
	}
}
