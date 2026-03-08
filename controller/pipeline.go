package controller

import (
	"errors"
	"sort"
	"strconv"

	"gateway/dao"
	"gateway/http_proxy_pipeline"
	"gateway/http_proxy_plugin"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

type pipelineInvalidateInput struct {
	ServiceID int64 `json:"service_id"`
}

func PipelineRegister(group *gin.RouterGroup) {
	group.GET("/plugins", PipelinePlugins)
	group.GET("/plan", PipelinePlan)
	group.GET("/cache", PipelineCache)
	group.POST("/invalidate", PipelineInvalidate)
}

func PipelinePlugins(c *gin.Context) {
	meta := http_proxy_plugin.GlobalRegistry.ListMeta()
	sort.Slice(meta, func(i, j int) bool {
		if meta[i].Phase != meta[j].Phase {
			return meta[i].Phase < meta[j].Phase
		}
		if meta[i].Priority != meta[j].Priority {
			return meta[i].Priority > meta[j].Priority
		}
		return meta[i].Name < meta[j].Name
	})
	middleware.ResponseSuccess(c, gin.H{
		"total":   len(meta),
		"plugins": meta,
	})
}

func PipelinePlan(c *gin.Context) {
	service, err := resolveServiceFromQuery(c)
	if err != nil {
		middleware.ResponseError(c, 4000, err)
		return
	}

	plan, err := http_proxy_pipeline.BuildPlanForService(c, service)
	if err != nil {
		middleware.ResponseError(c, 5000, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{
		"service_id":      plan.ServiceID,
		"service_name":    plan.ServiceName,
		"config_version":  plan.ConfigVersion,
		"plugins":         plan.Plugins,
		"warnings":        plan.Warnings,
		"cache_size":      len(http_proxy_pipeline.CachedPlans()),
		"requested_debug": c.Query("pipeline_debug") == "1",
	})
}

func PipelineCache(c *gin.Context) {
	plans := http_proxy_pipeline.CachedPlans()
	middleware.ResponseSuccess(c, gin.H{
		"total": len(plans),
		"plans": plans,
	})
}

func PipelineInvalidate(c *gin.Context) {
	var input pipelineInvalidateInput
	_ = c.ShouldBindJSON(&input)
	if input.ServiceID <= 0 {
		if raw := c.Query("service_id"); raw != "" {
			v, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				middleware.ResponseError(c, 4000, errors.New("invalid service_id"))
				return
			}
			input.ServiceID = v
		}
	}

	if input.ServiceID > 0 {
		http_proxy_pipeline.InvalidateService(input.ServiceID)
		middleware.ResponseSuccess(c, gin.H{
			"message":    "invalidate service plan success",
			"service_id": input.ServiceID,
		})
		return
	}

	http_proxy_pipeline.InvalidateAll()
	middleware.ResponseSuccess(c, gin.H{
		"message": "invalidate all plans success",
	})
}

func resolveServiceFromQuery(c *gin.Context) (*dao.ServiceDetail, error) {
	if raw := c.Query("service_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("invalid service_id")
		}
		service, ok := dao.ServiceManagerHandler.GetByID(id)
		if !ok {
			return nil, errors.New("service not found in runtime cache")
		}
		return service, nil
	}

	serviceName := c.Query("service_name")
	if serviceName == "" {
		return nil, errors.New("service_id or service_name is required")
	}
	service, ok := dao.ServiceManagerHandler.GetByName(serviceName)
	if !ok {
		return nil, errors.New("service not found in runtime cache")
	}
	return service, nil
}
