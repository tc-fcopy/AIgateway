package http_proxy_plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gateway/dao"
	"gateway/reverse_proxy/load_balance"
	"github.com/gin-gonic/gin"
)

type mockLoadBalance struct{}

func (m *mockLoadBalance) Add(...string) error {
	return nil
}

func (m *mockLoadBalance) Get(string) (string, error) {
	return "", nil
}

func (m *mockLoadBalance) Update() {}

type mockReverseProxy struct {
	served bool
}

func (m *mockReverseProxy) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	m.served = true
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("proxied"))
}

func TestProxyReverseProxyPluginServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareProxyReverseProxyTestState()
	defer restore()

	r := gin.New()
	r.Use(NewProxyReverseProxyPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "next")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "service not found") {
		t.Fatalf("expected service not found response, got %s", w.Body.String())
	}
}

func TestProxyReverseProxyPluginLoadBalancerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareProxyReverseProxyTestState()
	defer restore()

	serviceLoadBalancerGetter = func(*dao.ServiceDetail) (load_balance.LoadBalance, error) {
		return nil, errors.New("lb error")
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", &dao.ServiceDetail{})
		c.Next()
	})
	r.Use(NewProxyReverseProxyPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "next")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "lb error") {
		t.Fatalf("expected lb error response, got %s", w.Body.String())
	}
}

func TestProxyReverseProxyPluginTransportError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareProxyReverseProxyTestState()
	defer restore()

	serviceLoadBalancerGetter = func(*dao.ServiceDetail) (load_balance.LoadBalance, error) {
		return &mockLoadBalance{}, nil
	}
	serviceTransportGetter = func(*dao.ServiceDetail) (*http.Transport, error) {
		return nil, errors.New("transport error")
	}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", &dao.ServiceDetail{})
		c.Next()
	})
	r.Use(NewProxyReverseProxyPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "next")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "transport error") {
		t.Fatalf("expected transport error response, got %s", w.Body.String())
	}
}

func TestProxyReverseProxyPluginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareProxyReverseProxyTestState()
	defer restore()

	mockProxy := &mockReverseProxy{}
	serviceLoadBalancerGetter = func(*dao.ServiceDetail) (load_balance.LoadBalance, error) {
		return &mockLoadBalance{}, nil
	}
	serviceTransportGetter = func(*dao.ServiceDetail) (*http.Transport, error) {
		return &http.Transport{}, nil
	}
	reverseProxyBuilder = func(_ *gin.Context, _ load_balance.LoadBalance, _ *http.Transport) reverseProxyLike {
		return mockProxy
	}

	nextCalled := false
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", &dao.ServiceDetail{})
		c.Next()
	})
	r.Use(NewProxyReverseProxyPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		nextCalled = true
		c.String(http.StatusOK, "next")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 from proxy writer, got %d", w.Code)
	}
	if w.Body.String() != "proxied" {
		t.Fatalf("unexpected proxy response body: %s", w.Body.String())
	}
	if !mockProxy.served {
		t.Fatalf("expected reverse proxy ServeHTTP called")
	}
	if nextCalled {
		t.Fatalf("expected downstream handler not called after proxy abort")
	}
}

func prepareProxyReverseProxyTestState() func() {
	prevLBGetter := serviceLoadBalancerGetter
	prevTransGetter := serviceTransportGetter
	prevBuilder := reverseProxyBuilder
	return func() {
		serviceLoadBalancerGetter = prevLBGetter
		serviceTransportGetter = prevTransGetter
		reverseProxyBuilder = prevBuilder
	}
}
