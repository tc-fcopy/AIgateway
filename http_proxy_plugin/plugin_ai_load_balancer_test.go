package http_proxy_plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/loadbalancer"
	"github.com/gin-gonic/gin"
)

type mockAILoadBalancer struct {
	backend    *loadbalancer.Backend
	selectErr  error
	selectCall int
	released   []string
}

func (m *mockAILoadBalancer) SelectBackend() (*loadbalancer.Backend, error) {
	m.selectCall++
	if m.selectErr != nil {
		return nil, m.selectErr
	}
	return m.backend, nil
}

func (m *mockAILoadBalancer) ReleaseBackend(backendID string) {
	m.released = append(m.released, backendID)
}

func TestAILoadBalancerPluginSelectAndRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAILoadBalancerTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableLoadBalancer: true,
		},
	})

	mock := &mockAILoadBalancer{
		backend: &loadbalancer.Backend{ID: "b-1", Address: "10.0.0.1:8080"},
	}
	aiLoadBalancerGetter = func() aiLoadBalancerLike { return mock }

	var backendIDCtx string
	var backendAddrCtx string
	var backendIDHeader string
	var backendAddrHeader string

	r := gin.New()
	r.Use(NewAILoadBalancerPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		backendIDCtx = c.GetString(aigwctx.BackendIDKey)
		backendAddrCtx = c.GetString(aigwctx.BackendAddressKey)
		backendIDHeader = c.GetHeader("X-Backend-ID")
		backendAddrHeader = c.GetHeader("X-Backend-Address")
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if backendIDCtx != "b-1" || backendAddrCtx != "10.0.0.1:8080" {
		t.Fatalf("unexpected backend context: id=%s addr=%s", backendIDCtx, backendAddrCtx)
	}
	if backendIDHeader != "b-1" || backendAddrHeader != "10.0.0.1:8080" {
		t.Fatalf("unexpected backend headers: id=%s addr=%s", backendIDHeader, backendAddrHeader)
	}
	if mock.selectCall != 1 {
		t.Fatalf("expected select call once, got %d", mock.selectCall)
	}
	if len(mock.released) != 1 || mock.released[0] != "b-1" {
		t.Fatalf("expected release b-1 once, got %#v", mock.released)
	}
}

func TestAILoadBalancerPluginSelectErrorPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAILoadBalancerTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableLoadBalancer: true,
		},
	})

	mock := &mockAILoadBalancer{
		selectErr: errors.New("no backend"),
	}
	aiLoadBalancerGetter = func() aiLoadBalancerLike { return mock }

	var backendIDCtx string
	r := gin.New()
	r.Use(NewAILoadBalancerPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		backendIDCtx = c.GetString(aigwctx.BackendIDKey)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if backendIDCtx != "" {
		t.Fatalf("expected no backend context on select error, got %s", backendIDCtx)
	}
	if len(mock.released) != 0 {
		t.Fatalf("expected no release on select error, got %#v", mock.released)
	}
}

func prepareAILoadBalancerTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevGetter := aiLoadBalancerGetter
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiLoadBalancerGetter = prevGetter
	}
}
