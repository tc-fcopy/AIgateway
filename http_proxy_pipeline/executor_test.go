package http_proxy_pipeline

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	"gateway/dao"
	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
)

type stubPlugin struct {
	name     string
	phase    http_proxy_plugin.Phase
	priority int
	execFn   func(*http_proxy_plugin.ExecContext) http_proxy_plugin.Result
}

func (s *stubPlugin) Name() string { return s.name }
func (s *stubPlugin) Phase() http_proxy_plugin.Phase {
	return s.phase
}
func (s *stubPlugin) Priority() int { return s.priority }
func (s *stubPlugin) Requires() []string {
	return nil
}
func (s *stubPlugin) Enabled(*http_proxy_plugin.ExecContext) bool { return true }
func (s *stubPlugin) Execute(ctx *http_proxy_plugin.ExecContext) http_proxy_plugin.Result {
	if s.execFn == nil {
		return http_proxy_plugin.Continue()
	}
	return s.execFn(ctx)
}

func TestExecutorExecutesPlanOrderWithNativePlugins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := http_proxy_plugin.NewRegistry()
	var mu sync.Mutex
	order := make([]string, 0, 2)

	reg.MustRegister(&stubPlugin{
		name:     "test.a",
		phase:    http_proxy_plugin.PhasePreflight,
		priority: 100,
		execFn: func(*http_proxy_plugin.ExecContext) http_proxy_plugin.Result {
			mu.Lock()
			order = append(order, "a")
			mu.Unlock()
			return http_proxy_plugin.Continue()
		},
	})
	reg.MustRegister(&stubPlugin{
		name:     "test.b",
		phase:    http_proxy_plugin.PhaseProxy,
		priority: 10,
		execFn: func(*http_proxy_plugin.ExecContext) http_proxy_plugin.Result {
			mu.Lock()
			order = append(order, "b")
			mu.Unlock()
			return http_proxy_plugin.Continue()
		},
	})

	r := gin.New()
	r.Use(injectServiceAndPlan([]string{"test.a", "test.b"}))
	r.Use(NewExecutor(reg).Middleware())
	r.Any("/*any", func(c *gin.Context) {
		c.String(299, "tail")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/chat", nil)
	r.ServeHTTP(w, req)

	if got := w.Code; got != 200 {
		t.Fatalf("expected status 200, got %d", got)
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("unexpected execution order: %#v", order)
	}
}

func TestExecutorAbortFromPlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := http_proxy_plugin.NewRegistry()
	reg.MustRegister(&stubPlugin{
		name:     "test.abort",
		phase:    http_proxy_plugin.PhasePolicy,
		priority: 100,
		execFn: func(*http_proxy_plugin.ExecContext) http_proxy_plugin.Result {
			return http_proxy_plugin.AbortWithCode(429, 5002, "rate limit", nil)
		},
	})

	r := gin.New()
	r.Use(injectServiceAndPlan([]string{"test.abort"}))
	r.Use(NewExecutor(reg).Middleware())
	r.Any("/*any", func(c *gin.Context) {
		c.String(299, "tail")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/chat", nil)
	r.ServeHTTP(w, req)

	if got := w.Code; got != 429 {
		t.Fatalf("expected status 429, got %d", got)
	}

	body := map[string]interface{}{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}
	if body["errno"] != float64(5002) {
		t.Fatalf("expected errno 5002, got %#v", body["errno"])
	}
}

func TestExecutorPreservesLegacyMiddlewareNextSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reg := http_proxy_plugin.NewRegistry()
	order := make([]string, 0, 4)

	p1, err := http_proxy_plugin.NewMiddlewareAdapter(http_proxy_plugin.AdapterSpec{
		Name:     "legacy.a",
		Phase:    http_proxy_plugin.PhaseTransform,
		Priority: 100,
	}, func(c *gin.Context) {
		order = append(order, "a.pre")
		c.Next()
		order = append(order, "a.post")
	})
	if err != nil {
		t.Fatalf("create adapter plugin failed: %v", err)
	}
	p2, err := http_proxy_plugin.NewMiddlewareAdapter(http_proxy_plugin.AdapterSpec{
		Name:     "legacy.b",
		Phase:    http_proxy_plugin.PhaseTransform,
		Priority: 90,
	}, func(c *gin.Context) {
		order = append(order, "b.pre")
		c.Next()
		order = append(order, "b.post")
	})
	if err != nil {
		t.Fatalf("create adapter plugin failed: %v", err)
	}
	reg.MustRegister(p1)
	reg.MustRegister(p2)

	r := gin.New()
	r.Use(injectServiceAndPlan([]string{"legacy.a", "legacy.b"}))
	r.Use(NewExecutor(reg).Middleware())
	r.Any("/*any", func(c *gin.Context) {
		c.String(299, "tail")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/chat", nil)
	r.ServeHTTP(w, req)

	expected := []string{"a.pre", "b.pre", "b.post", "a.post"}
	if len(order) != len(expected) {
		t.Fatalf("unexpected order length: %#v", order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("unexpected order at %d: want=%s got=%s", i, expected[i], order[i])
		}
	}
}

func injectServiceAndPlan(plugins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 1, ServiceName: "svc-test"}}
		c.Set("service", service)
		c.Set("service_name", service.Info.ServiceName)
		c.Set("service_id", service.Info.ID)
		c.Set(CtxPlanKey, buildTestPlan(service.Info.ID, service.Info.ServiceName, plugins))
		c.Next()
	}
}

func buildTestPlan(serviceID int64, serviceName string, plugins []string) *Plan {
	set := make(map[string]struct{}, len(plugins))
	for _, p := range plugins {
		set[p] = struct{}{}
	}
	return &Plan{
		ServiceID:     serviceID,
		ServiceName:   serviceName,
		ConfigVersion: "test",
		Plugins:       plugins,
		Warnings:      nil,
		pluginSet:     set,
	}
}
