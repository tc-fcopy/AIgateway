package http_proxy_plugin

import (
	"net/http/httptest"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestProxyURLRewritePluginRewritePath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)
	c.Set("service", &dao.ServiceDetail{
		HTTPRule: &dao.HttpRule{
			UrlRewrite: "^/api /internal,^/internal/v1 /upstream/v1",
		},
	})

	plugin := NewProxyURLRewritePlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	if got := c.Request.URL.Path; got != "/upstream/v1/chat/completions" {
		t.Fatalf("unexpected rewritten path: %s", got)
	}
}

func TestProxyURLRewritePluginInvalidRuleIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)
	c.Set("service", &dao.ServiceDetail{
		HTTPRule: &dao.HttpRule{
			UrlRewrite: "[invalid /internal,broken_rule",
		},
	})

	plugin := NewProxyURLRewritePlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	if got := c.Request.URL.Path; got != "/api/v1/chat/completions" {
		t.Fatalf("expected unchanged path, got %s", got)
	}
}

func TestProxyURLRewritePluginServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)

	plugin := NewProxyURLRewritePlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort result when service missing")
	}
	if !c.IsAborted() {
		t.Fatalf("expected gin context aborted")
	}
}
