package http_proxy_pipeline

import (
	"net/http/httptest"
	"strings"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	"gateway/dao"
	"github.com/gin-gonic/gin"
)

func TestPipelinePlanMiddleware_AIEnabledByApplyAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: true,
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/v1/chat?pipeline_debug=1", nil)
	c.Request = req
	c.Set("service", &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 101, ServiceName: "svc-ai"}})

	PipelinePlanMiddleware()(c)

	plan, ok := GetPlan(c)
	if !ok || plan == nil {
		t.Fatalf("expected pipeline plan in context")
	}
	if !plan.Has(PluginAIAuth) {
		t.Fatalf("expected %s to be enabled when apply_to_all_services=true", PluginAIAuth)
	}
	if got := w.Header().Get("X-Pipeline-Plugins"); !strings.Contains(got, PluginAIAuth) {
		t.Fatalf("expected debug header to include %s, got: %s", PluginAIAuth, got)
	}
}

func TestPipelinePlanMiddleware_AIDisabledWhenApplyAllFalseAndNoServiceOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(false),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: true,
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/v1/chat?pipeline_debug=1", nil)
	c.Request = req
	c.Set("service", &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 102, ServiceName: "svc-normal"}})

	PipelinePlanMiddleware()(c)

	plan, ok := GetPlan(c)
	if !ok || plan == nil {
		t.Fatalf("expected pipeline plan in context")
	}
	if plan.Has(PluginAIAuth) {
		t.Fatalf("did not expect %s when apply_to_all_services=false and no service override", PluginAIAuth)
	}
	if !plan.Has(PluginCoreFlowLimit) {
		t.Fatalf("expected core middleware to remain enabled")
	}
	if ShouldExecute(c, PluginAIAuth) {
		t.Fatalf("expected ShouldExecute(%s)=false", PluginAIAuth)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
