package quota

import "testing"

func TestParseInt64(t *testing.T) {
	if v := parseInt64("abc123x"); v != 123 {
		t.Fatalf("expected 123, got %d", v)
	}
}

func TestSetConfig(t *testing.T) {
	m := NewManager()
	m.SetConfig(false, 200, 3600)
	if m.IsEnabled() {
		t.Fatalf("expected disabled manager")
	}
	if m.GetDefaultQuota() != 200 {
		t.Fatalf("expected default quota 200, got %d", m.GetDefaultQuota())
	}
	if m.GetQuotaTTL() != 3600 {
		t.Fatalf("expected ttl 3600, got %d", m.GetQuotaTTL())
	}
}
