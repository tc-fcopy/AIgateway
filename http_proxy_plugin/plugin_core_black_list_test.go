package http_proxy_plugin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestCoreBlackListPluginExecuteDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreBlackListPlugin()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.9:1234"
	c.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: "svc-bl"},
		AccessControl: &dao.AccessControl{
			OpenAuth:  1,
			WhiteList: "",
			BlackList: "203.0.113.9,192.0.2.10",
		},
	})

	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected request denied by blacklist")
	}
	if !strings.Contains(w.Body.String(), "in black ip list") {
		t.Fatalf("expected blacklist denial message, got: %s", w.Body.String())
	}
}

func TestCoreBlackListPluginExecuteAllowedWhenWhitelistConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreBlackListPlugin()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.9:1234"
	c.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: "svc-bl"},
		AccessControl: &dao.AccessControl{
			OpenAuth:  1,
			WhiteList: "198.51.100.1",
			BlackList: "203.0.113.9",
		},
	})

	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected blacklist plugin bypass when whitelist exists")
	}
}

func TestCoreBlackListPluginExecuteServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreBlackListPlugin()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort when service missing")
	}
	if !strings.Contains(w.Body.String(), "service not found") {
		t.Fatalf("expected service missing response, got: %s", w.Body.String())
	}
}
