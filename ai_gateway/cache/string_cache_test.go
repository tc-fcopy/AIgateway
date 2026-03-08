package cache

import "testing"

func TestGenerateCacheKeyFromRequest(t *testing.T) {
	c := NewStringCache(nil)
	body := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`)
	k1, err := c.GenerateCacheKeyFromRequest("consumer-a", "gpt-4", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	k2, err := c.GenerateCacheKeyFromRequest("consumer-a", "gpt-4", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k1 != k2 {
		t.Fatalf("expected deterministic key, got %s and %s", k1, k2)
	}
}
