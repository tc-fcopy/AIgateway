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

// syncServiceRuntime refreshes route matching cache and invalidates pipeline plan cache.
func syncServiceRuntime(c *gin.Context, serviceID int64) error {
	if err := serviceReloadFunc(); err != nil {
		return fmt.Errorf("reload service manager failed: %w", err)
	}
	return syncAIServiceConfigRuntime(serviceID)
}

// syncAIServiceConfigRuntime invalidates pipeline plan cache for AI service config updates.
func syncAIServiceConfigRuntime(serviceID int64) error {
	if err := reloadAIServiceRuntimeFunc(serviceID); err != nil {
		return fmt.Errorf("reload ai service config runtime failed: %w", err)
	}

	if serviceID > 0 {
		invalidateServicePlanFunc(serviceID)
	} else {
		invalidateAllPlansFunc()
	}
	return nil
}
