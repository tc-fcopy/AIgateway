package http_proxy_plugin

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gateway/dao"
	"gateway/public"
	"github.com/gin-gonic/gin"
)

func TestCoreFlowLimitPluginExecuteServiceFlowLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := public.FlowLimiterHandler
	public.FlowLimiterHandler = public.NewFlowLimiter()
	defer func() {
		public.FlowLimiterHandler = orig
	}()

	serviceName := fmt.Sprintf("svc-limit-%d", time.Now().UnixNano())
	plugin := NewCoreFlowLimitPlugin()

	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("GET", "/", nil)
	c1.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: serviceName},
		AccessControl: &dao.AccessControl{
			ServiceFlowLimit: 1,
		},
	})
	r1 := plugin.Execute(NewExecContext(c1))
	if r1.IsAbort() {
		t.Fatalf("expected first request pass, got abort")
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Set("service", &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ServiceName: serviceName},
		AccessControl: &dao.AccessControl{
			ServiceFlowLimit: 1,
		},
	})
	r2 := plugin.Execute(NewExecContext(c2))
	if !r2.IsAbort() {
		t.Fatalf("expected second request limited")
	}
	if !strings.Contains(w2.Body.String(), "service flow limit") {
		t.Fatalf("expected flow limit message, got: %s", w2.Body.String())
	}
}

func TestCoreFlowLimitPluginExecuteServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	plugin := NewCoreFlowLimitPlugin()
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
