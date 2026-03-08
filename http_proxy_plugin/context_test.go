package http_proxy_plugin

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewExecContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("service_id", int64(11))
	c.Set("service_name", "svc-a")

	exec := NewExecContext(c)
	if exec.ServiceID != 11 {
		t.Fatalf("expected service id 11, got %d", exec.ServiceID)
	}
	if exec.ServiceName != "svc-a" {
		t.Fatalf("expected service name svc-a, got %s", exec.ServiceName)
	}

	exec.SetValue("k", "v")
	v, ok := exec.GetValue("k")
	if !ok || v.(string) != "v" {
		t.Fatalf("context value mismatch")
	}
}
