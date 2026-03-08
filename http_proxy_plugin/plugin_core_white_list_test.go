package http_proxy_plugin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestCoreWhiteListPluginExecuteAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreWhiteListPlugin()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "192.0.2.10:3456"
	c.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: "svc-wl"},
		AccessControl: &dao.AccessControl{
			OpenAuth:  1,
			WhiteList: "192.0.2.10,198.51.100.1",
		},
	})

	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected request allowed by whitelist")
	}
}

func TestCoreWhiteListPluginExecuteDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreWhiteListPlugin()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.9:1234"
	c.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: "svc-wl"},
		AccessControl: &dao.AccessControl{
			OpenAuth:  1,
			WhiteList: "192.0.2.10,198.51.100.1",
		},
	})

	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected request denied by whitelist")
	}
	if !strings.Contains(w.Body.String(), "not in white ip list") {
		t.Fatalf("expected whitelist denial message, got: %s", w.Body.String())
	}
}
