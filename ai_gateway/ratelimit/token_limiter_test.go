package ratelimit

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTimeWindows(t *testing.T) {
	l := NewTokenLimiter(nil, true)
	w := l.getTimeWindows()
	if len(w) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(w))
	}
}

func TestGetLimitDefaults(t *testing.T) {
	l := NewTokenLimiter(nil, true)
	limit, ok := l.getLimit(&gin.Context{})
	if !ok || limit <= 0 {
		t.Fatalf("expected positive default limit")
	}
}
