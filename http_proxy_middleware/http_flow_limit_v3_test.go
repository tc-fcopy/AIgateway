package http_proxy_middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gateway/dao"
	"gateway/public"
	"github.com/gin-gonic/gin"
)

func TestHTTPFlowLimitMiddlewareV3_MissingService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(HTTPFlowLimitMiddlewareV3())
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "service not found") {
		t.Fatalf("expected business error in response body, got: %s", w.Body.String())
	}
}

func TestHTTPFlowLimitMiddlewareV3_ServiceFlowLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	serviceName := fmt.Sprintf("svc_%d", time.Now().UnixNano())
	// Reset global limiter to avoid cross-test interference.
	public.FlowLimiterHandler = public.NewFlowLimiter()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("service", &dao.ServiceDetail{
			Info: &dao.ServiceInfo{ServiceName: serviceName},
			AccessControl: &dao.AccessControl{
				ServiceFlowLimit: 1,
			},
		})
		c.Next()
	})
	r.Use(HTTPFlowLimitMiddlewareV3())
	r.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request should pass.
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected first request to pass, got %d", w1.Code)
	}

	// Immediate second request should be limited.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w2, req2)
	if !strings.Contains(w2.Body.String(), "flow limit") {
		t.Fatalf("expected second request to be limited, got %s", w2.Body.String())
	}
}
