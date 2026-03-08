package http_proxy_plugin

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"gateway/dao"
	"gateway/public"
	"github.com/gin-gonic/gin"
)

func TestCoreFlowCountPluginExecute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uniqueService := fmt.Sprintf("svc-%d", time.Now().UnixNano())
	serviceKey := public.FlowServicePrefix + uniqueService

	totalCounter, err := public.FlowCounterHandler.GetCounter(public.FlowTotal)
	if err != nil {
		t.Fatalf("get total counter failed: %v", err)
	}
	serviceCounter, err := public.FlowCounterHandler.GetCounter(serviceKey)
	if err != nil {
		t.Fatalf("get service counter failed: %v", err)
	}
	beforeTotal := totalCounter.TotalCount
	beforeService := serviceCounter.TotalCount

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("service", &dao.ServiceDetail{Info: &dao.ServiceInfo{ID: 1, ServiceName: uniqueService}})

	plugin := NewCoreFlowCountPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result, got abort")
	}

	if totalCounter.TotalCount != beforeTotal+1 {
		t.Fatalf("expected total counter +1, before=%d after=%d", beforeTotal, totalCounter.TotalCount)
	}
	if serviceCounter.TotalCount != beforeService+1 {
		t.Fatalf("expected service counter +1, before=%d after=%d", beforeService, serviceCounter.TotalCount)
	}
}

func TestCoreFlowCountPluginExecuteServiceMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	plugin := NewCoreFlowCountPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort result when service context missing")
	}
}
