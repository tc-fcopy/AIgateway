package http_proxy_middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestHTTPFlowLimitMiddlewareV3_ClientIPLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", &dao.ServiceDetail{
			Info: &dao.ServiceInfo{ServiceName: "ip_limiter"},
			AccessControl: &dao.AccessControl{
				ClientIPFlowLimit: 1,
			},
		})
		c.Next()
	})
	r.Use(HTTPFlowLimitMiddlewareV3())
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	r.ServeHTTP(w2, req2)
	if w2.Code == http.StatusOK {
		t.Fatalf("expected second request to be limited")
	}
}
