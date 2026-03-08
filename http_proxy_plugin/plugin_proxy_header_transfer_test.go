package http_proxy_plugin

import (
	"net/http/httptest"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestProxyHeaderTransferPluginApplyRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Edit", "old")
	c.Request.Header.Set("X-Del", "to-delete")
	c.Set("service", &dao.ServiceDetail{
		HTTPRule: &dao.HttpRule{
			HeaderTransfor: "add X-Add 1,edit X-Edit 2,del X-Del x,broken_rule",
		},
	})

	plugin := NewProxyHeaderTransferPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	if got := c.Request.Header.Get("X-Add"); got != "1" {
		t.Fatalf("expected X-Add=1, got %s", got)
	}
	if got := c.Request.Header.Get("X-Edit"); got != "2" {
		t.Fatalf("expected X-Edit=2, got %s", got)
	}
	if got := c.Request.Header.Get("X-Del"); got != "" {
		t.Fatalf("expected X-Del deleted, got %s", got)
	}
}

func TestProxyHeaderTransferPluginServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	plugin := NewProxyHeaderTransferPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort result when service missing")
	}
	if !c.IsAborted() {
		t.Fatalf("expected gin context aborted")
	}
}

func TestProxyHeaderTransferPluginServiceInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("service", &dao.ServiceDetail{})

	plugin := NewProxyHeaderTransferPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort result when service invalid")
	}
	if !c.IsAborted() {
		t.Fatalf("expected gin context aborted")
	}
}
