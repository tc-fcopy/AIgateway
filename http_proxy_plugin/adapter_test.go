package http_proxy_plugin

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMiddlewareAdapter_Continue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adapter, err := NewMiddlewareAdapter(AdapterSpec{
		Name:     "test.continue",
		Phase:    PhasePolicy,
		Priority: 10,
	}, func(c *gin.Context) {
		c.Set("ran", true)
	})
	if err != nil {
		t.Fatalf("new adapter error: %v", err)
	}

	res := adapter.Execute(NewExecContext(c))
	if res.IsAbort() {
		t.Fatalf("expected continue result, got abort: %+v", res)
	}
	if v := c.GetBool("ran"); !v {
		t.Fatalf("expected adapted middleware to run")
	}
}

func TestMiddlewareAdapter_Abort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adapter, err := NewMiddlewareAdapter(AdapterSpec{
		Name:  "test.abort",
		Phase: PhasePolicy,
	}, func(c *gin.Context) {
		c.AbortWithStatus(403)
	})
	if err != nil {
		t.Fatalf("new adapter error: %v", err)
	}

	res := adapter.Execute(NewExecContext(c))
	if !res.IsAbort() {
		t.Fatalf("expected abort result")
	}
	if res.HTTPStatus != 403 {
		t.Fatalf("expected status 403, got %d", res.HTTPStatus)
	}
}

func TestMiddlewareAdapter_Validation(t *testing.T) {
	if _, err := NewMiddlewareAdapter(AdapterSpec{}, func(c *gin.Context) {}); err == nil {
		t.Fatalf("expected error for empty name")
	}
	if _, err := NewMiddlewareAdapter(AdapterSpec{Name: "x"}, nil); err == nil {
		t.Fatalf("expected error for nil handler")
	}
}
