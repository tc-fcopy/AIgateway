package http_proxy_plugin

import (
	"net/http/httptest"
	"testing"

	"gateway/dao"
	"gateway/public"
	"github.com/gin-gonic/gin"
)

func TestProxyStripURIPluginStripPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)
	c.Set("service", &dao.ServiceDetail{
		HTTPRule: &dao.HttpRule{
			RuleType:     public.HTTPRuleTypePrefixURL,
			NeedStripUri: 1,
			Rule:         "/api",
		},
	})

	plugin := NewProxyStripURIPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}
	if got := c.Request.URL.Path; got != "/v1/chat/completions" {
		t.Fatalf("expected stripped path, got %s", got)
	}
}

func TestProxyStripURIPluginNoStripWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)
	c.Set("service", &dao.ServiceDetail{
		HTTPRule: &dao.HttpRule{
			RuleType:     public.HTTPRuleTypePrefixURL,
			NeedStripUri: 0,
			Rule:         "/api",
		},
	})

	plugin := NewProxyStripURIPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}
	if got := c.Request.URL.Path; got != "/api/v1/chat/completions" {
		t.Fatalf("expected unchanged path, got %s", got)
	}
}

func TestProxyStripURIPluginServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/chat/completions", nil)

	plugin := NewProxyStripURIPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort result when service missing")
	}
	if !c.IsAborted() {
		t.Fatalf("expected gin context aborted")
	}
}
