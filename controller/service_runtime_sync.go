package controller

import (
	"fmt"

	"gateway/dao"
	"gateway/http_proxy_pipeline"
	"github.com/gin-gonic/gin"
)

var (
	serviceReloadFunc = func() error {
		return dao.ServiceManagerHandler.Reload()
	}
	reloadAIServiceRuntimeFunc = func(serviceID int64) error {
		return http_proxy_pipeline.ReloadAIServiceConfigRuntime(serviceID)
	}
	invalidateServicePlanFunc = func(serviceID int64) {
		http_proxy_pipeline.InvalidateService(serviceID)
	}
	invalidateAllPlansFunc = func() {
		http_proxy_pipeline.InvalidateAll()
	}
)

// syncServiceRuntime refreshes route matching cache and prebuilds pipeline plans.
func syncServiceRuntime(c *gin.Context, serviceID int64) error {
	if err := serviceReloadFunc(); err != nil {
		return fmt.Errorf("reload service manager failed: %w", err)
	}
	return syncAIServiceConfigRuntime(serviceID)
}

// syncAIServiceConfigRuntime prebuilds pipeline plans for AI service config updates.
func syncAIServiceConfigRuntime(serviceID int64) error {
	if err := reloadAIServiceRuntimeFunc(serviceID); err != nil {
		return fmt.Errorf("reload ai service config runtime failed: %w", err)
	}

	if serviceID > 0 {
		service, ok := dao.ServiceManagerHandler.GetByID(serviceID)
		if !ok || service == nil {
			invalidateServicePlanFunc(serviceID)
			return nil
		}
		if _, err := http_proxy_pipeline.PrebuildPlanForService(service); err != nil {
			invalidateServicePlanFunc(serviceID)
			return fmt.Errorf("prebuild pipeline plan failed: %w", err)
		}
		return nil
	}

	services := dao.ServiceManagerHandler.List()
	if err := http_proxy_pipeline.PrebuildPlans(services); err != nil {
		invalidateAllPlansFunc()
		return fmt.Errorf("prebuild pipeline plans failed: %w", err)
	}
	return nil
}
