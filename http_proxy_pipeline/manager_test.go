package http_proxy_pipeline

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	aiconfig "gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/http_proxy_plugin"
)

func TestBuildPlanContext_AIServiceConfigRuntimeCached(t *testing.T) {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevBulk := aiServiceConfigBulkLoader
	prevSingle := aiServiceConfigSingleLoader
	defer func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiServiceConfigBulkLoader = prevBulk
		aiServiceConfigSingleLoader = prevSingle
		resetAIServiceConfigRuntimeForTest()
	}()

	resetAIServiceConfigRuntimeForTest()
	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(false),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: false,
		},
	})

	var bulkCalls int32
	aiServiceConfigBulkLoader = func() ([]dao.AIServiceConfig, error) {
		atomic.AddInt32(&bulkCalls, 1)
		return []dao.AIServiceConfig{
			{ServiceID: 7, EnableKeyAuth: true},
		}, nil
	}
	aiServiceConfigSingleLoader = func(serviceID int64) (*dao.AIServiceConfig, error) {
		t.Fatalf("unexpected single loader call for service=%d", serviceID)
		return nil, nil
	}

	service := &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 7, ServiceName: "svc-7"}}

	pc1, err := BuildPlanContext(nil, service)
	if err != nil {
		t.Fatalf("BuildPlanContext returned error: %v", err)
	}
	if !pc1.AIEnabled || !pc1.EnableAuth {
		t.Fatalf("expected AI service override to enable auth, got AIEnabled=%v EnableAuth=%v", pc1.AIEnabled, pc1.EnableAuth)
	}

	pc2, err := BuildPlanContext(nil, service)
	if err != nil {
		t.Fatalf("BuildPlanContext returned error: %v", err)
	}
	if !pc2.AIEnabled || !pc2.EnableAuth {
		t.Fatalf("expected AI service override to enable auth on second call")
	}
	if got := atomic.LoadInt32(&bulkCalls); got != 1 {
		t.Fatalf("expected bulk loader called once, got %d", got)
	}
}

func TestReloadAIServiceConfigRuntime_OneService(t *testing.T) {
	prevBulk := aiServiceConfigBulkLoader
	prevSingle := aiServiceConfigSingleLoader
	defer func() {
		aiServiceConfigBulkLoader = prevBulk
		aiServiceConfigSingleLoader = prevSingle
		resetAIServiceConfigRuntimeForTest()
	}()

	resetAIServiceConfigRuntimeForTest()

	var bulkCalls int32
	var singleCalls int32
	aiServiceConfigBulkLoader = func() ([]dao.AIServiceConfig, error) {
		atomic.AddInt32(&bulkCalls, 1)
		return []dao.AIServiceConfig{}, nil
	}
	aiServiceConfigSingleLoader = func(serviceID int64) (*dao.AIServiceConfig, error) {
		atomic.AddInt32(&singleCalls, 1)
		return &dao.AIServiceConfig{
			ServiceID:   serviceID,
			EnableCache: true,
		}, nil
	}

	if err := ReloadAIServiceConfigRuntime(0); err != nil {
		t.Fatalf("reload all failed: %v", err)
	}
	if err := ReloadAIServiceConfigRuntime(9); err != nil {
		t.Fatalf("reload one failed: %v", err)
	}

	row, ok, err := aiServiceConfigRuntimeCache.Get(9)
	if err != nil {
		t.Fatalf("runtime get failed: %v", err)
	}
	if !ok || row == nil {
		t.Fatalf("expected runtime cache row for service 9")
	}
	if !row.EnableCache {
		t.Fatalf("expected cache flag true in runtime row")
	}
	if got := atomic.LoadInt32(&bulkCalls); got != 1 {
		t.Fatalf("expected bulk loader called once, got %d", got)
	}
	if got := atomic.LoadInt32(&singleCalls); got != 1 {
		t.Fatalf("expected single loader called once, got %d", got)
	}
}

func TestPlannerBuild_SingleflightDedup(t *testing.T) {
	p := NewPlanner(nil)
	p.specs = []PluginSpec{
		{
			Name:     "test.plugin",
			Phase:    PhaseProxy,
			Priority: 1,
			Enabled:  alwaysOn,
		},
	}

	var buildCalls int32
	p.buildFn = func(serviceID int64, serviceName string, pc *PlanContext, specs []PluginSpec, registry *http_proxy_plugin.Registry) *Plan {
		atomic.AddInt32(&buildCalls, 1)
		time.Sleep(40 * time.Millisecond)
		return buildPlan(serviceID, serviceName, pc, specs, registry)
	}

	service := &dao.ServiceDetail{
		Info: &dao.ServiceInfo{ID: 0, ServiceName: "svc-singleflight"},
	}

	start := make(chan struct{})
	wg := sync.WaitGroup{}
	errCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := p.Build(nil, service)
			errCh <- err
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("unexpected build error: %v", err)
		}
	}

	if got := atomic.LoadInt32(&buildCalls); got != 1 {
		t.Fatalf("expected one build call with singleflight dedup, got %d", got)
	}
}
