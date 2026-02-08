package cache

import (
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to be found")
	}
	if val != "value1" {
		t.Errorf("got %v, want value1", val)
	}
}

func TestGetMiss(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected miss for nonexistent key")
	}
}

func TestDelete(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("key1", "value1")
	c.Delete("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}
}

func TestClear(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	c.Clear()
	if c.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", c.Size())
	}
}

func TestSize(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	if c.Size() != 0 {
		t.Errorf("expected size 0, got %d", c.Size())
	}

	c.Set("a", 1)
	c.Set("b", 2)
	if c.Size() != 2 {
		t.Errorf("expected size 2, got %d", c.Size())
	}
}

func TestExpiration(t *testing.T) {
	c := New(50 * time.Millisecond)
	defer c.Stop()

	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("expected key1 to exist immediately")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = c.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestHashKey(t *testing.T) {
	k1 := HashKey("a", "b", "c")
	k2 := HashKey("a", "b", "c")
	k3 := HashKey("x", "y", "z")

	if k1 != k2 {
		t.Error("same inputs should produce same hash")
	}
	if k1 == k3 {
		t.Error("different inputs should produce different hash")
	}
	if len(k1) != 64 {
		t.Errorf("SHA256 hex should be 64 chars, got %d", len(k1))
	}
}

func TestFileCache(t *testing.T) {
	fc := NewFileCache(5 * time.Minute)
	defer fc.Stop()

	fc.Set("/path/to/file.txt", []byte("hello world"))

	content, ok := fc.Get("/path/to/file.txt")
	if !ok {
		t.Fatal("expected file to be cached")
	}
	if string(content) != "hello world" {
		t.Errorf("got %q, want %q", string(content), "hello world")
	}

	_, ok = fc.Get("/nonexistent")
	if ok {
		t.Error("expected miss for uncached path")
	}

	fc.Clear()
	_, ok = fc.Get("/path/to/file.txt")
	if ok {
		t.Error("expected miss after clear")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			c.Set(key, n)
			c.Get(key)
			c.Size()
		}(i)
	}
	wg.Wait()
}

func TestOverwrite(t *testing.T) {
	c := New(5 * time.Minute)
	defer c.Stop()

	c.Set("key", "v1")
	c.Set("key", "v2")

	val, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "v2" {
		t.Errorf("got %v, want v2", val)
	}
	if c.Size() != 1 {
		t.Errorf("expected size 1, got %d", c.Size())
	}
}
