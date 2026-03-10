package http_proxy_pipeline

import (
	"net/http/httptest"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// BenchmarkPipelinePlanWithoutCache 测试第一次构建 Plan（无缓存）
func BenchmarkPipelinePlanWithoutCache(b *testing.B) {
	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableIPRestriction:   true,
			EnableTokenRateLimit:  true,
			EnableQuota:           true,
			EnableCache:           true,
			EnableLoadBalancer:    true,
			EnableObservability:   true,
			EnableModelRouter:     true,
			EnablePromptDecorator: true,
		},
	})

	p := NewPlanner(nil)
	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 999, ServiceName: "svc-bench"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次都清除缓存，模拟第一次构建
		p.InvalidateAll()
		_, _ = p.Build(nil, service)
	}
}

// BenchmarkPipelinePlanWithCache 测试从缓存获取 Plan（正常场景）
func BenchmarkPipelinePlanWithCache(b *testing.B) {
	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableIPRestriction:   true,
			EnableTokenRateLimit:  true,
			EnableQuota:           true,
			EnableCache:           true,
			EnableLoadBalancer:    true,
			EnableObservability:   true,
		},
	})

	p := NewPlanner(nil)
	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 999, ServiceName: "svc-bench"}}

	// 预热缓存
	_, _ = p.Build(nil, service)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Build(nil, service)
	}
}

// BenchmarkExecutorWithCompiledHandler 测试使用预编译执行链（优化后的方式）
func BenchmarkExecutorWithCompiledHandler(b *testing.B) {
	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableIPRestriction:   true,
			EnableTokenRateLimit:  true,
			EnableQuota:           true,
		},
	})

	p := NewPlanner(nil)
	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 999, ServiceName: "svc-bench"}}
	plan, _ := p.Build(nil, service)

	executor := NewExecutor(http_proxy_plugin.GlobalRegistry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("service", service)
		c.Set(CtxPlanKey, plan)
		req := httptest.NewRequest("GET", "/v1/chat", nil)
		c.Request = req

		executor.Middleware()(c)
	}
}

// BenchmarkFullRequestFlow 测试完整请求流程（从 router 到 Plan 到 执行）
func BenchmarkFullRequestFlow(b *testing.B) {
	prev := aiconfig.AIConfManager.GetConfig()
	defer aiconfig.AIConfManager.SetConfig(prev)

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableIPRestriction:   true,
			EnableTokenRateLimit:  true,
			EnableQuota:           true,
		},
	})

	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 999, ServiceName: "svc-bench"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("service", service)
		req := httptest.NewRequest("GET", "/v1/chat", nil)
		c.Request = req

		// 执行完整流程
		PipelinePlanMiddleware()(c)
		if !c.IsAborted() {
			PipelineExecutorMiddleware()(c)
		}
	}
}
