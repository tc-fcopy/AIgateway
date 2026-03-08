package http_proxy_pipeline

import (
	"strings"

	"gateway/dao"
	"github.com/gin-gonic/gin"
)

const (
	CtxPlanKey       = "pipeline_plan"
	CtxPlanPluginKey = "pipeline_plugins"
)

// PipelinePlanMiddleware builds a service-specific execution plan and stores it in context.
func PipelinePlanMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceRaw, ok := c.Get("service")
		if !ok {
			c.Next()
			return
		}

		service, ok := serviceRaw.(*dao.ServiceDetail)
		if !ok || service == nil || service.Info == nil {
			c.Next()
			return
		}

		// Normalize common service context for downstream policies.
		c.Set("service_name", service.Info.ServiceName)
		c.Set("service_id", service.Info.ID)

		plan, err := defaultPlanner.Build(c, service)
		if err == nil && plan != nil {
			plugins := strings.Join(plan.Plugins, ",")
			warnings := strings.Join(plan.Warnings, ";")

			c.Set(CtxPlanKey, plan)
			c.Set(CtxPlanPluginKey, plugins)
			if warnings != "" {
				c.Set("pipeline_warnings", warnings)
			}

			if isDebugPlan(c) {
				c.Header("X-Pipeline-Service", plan.ServiceName)
				c.Header("X-Pipeline-Plugins", plugins)
				c.Header("X-Pipeline-Version", plan.ConfigVersion)
				if warnings != "" {
					c.Header("X-Pipeline-Warnings", warnings)
				}
			}
		}

		c.Next()
	}
}

func GetPlan(c *gin.Context) (*Plan, bool) {
	v, ok := c.Get(CtxPlanKey)
	if !ok {
		return nil, false
	}
	p, ok := v.(*Plan)
	return p, ok
}

// ShouldExecute returns whether a plugin should execute for this request.
// If no plan is available, it returns true for backward compatibility.
func ShouldExecute(c *gin.Context, pluginName string) bool {
	plan, ok := GetPlan(c)
	if !ok || plan == nil {
		return true
	}
	return plan.Has(pluginName)
}

func isDebugPlan(c *gin.Context) bool {
	if c.Query("pipeline_debug") == "1" {
		return true
	}
	v := strings.TrimSpace(c.GetHeader("X-Pipeline-Debug"))
	return v == "1" || strings.EqualFold(v, "true")
}
