package http_proxy_pipeline

import (
	"fmt"
	"net/http"

	"gateway/dao"
	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
)

const (
	CtxExecutorEnabled = "pipeline_executor_enabled"
)

// MiddlewarePlugin is an optional plugin capability that exposes native gin middleware.
// Legacy middleware adapters implement this and rely on c.Next semantics.
type MiddlewarePlugin interface {
	Handler() gin.HandlerFunc
}

// Executor drives request execution based on compiled plan.
type Executor struct {
	registry *http_proxy_plugin.Registry
}

var defaultExecutor = NewExecutor(http_proxy_plugin.GlobalRegistry)

func NewExecutor(registry *http_proxy_plugin.Registry) *Executor {
	return &Executor{registry: registry}
}

// PipelineExecutorMiddleware executes plugins from the compiled plan.
func PipelineExecutorMiddleware() gin.HandlerFunc {
	return defaultExecutor.Middleware()
}

func (e *Executor) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		service, ok := getServiceFromContext(c)
		if !ok {
			c.Next()
			return
		}

		plan, err := e.getOrBuildPlan(c, service)
		if err != nil {
			writeExecutorError(c, http.StatusInternalServerError, 5000, err.Error())
			return
		}
		if plan == nil || len(plan.Plugins) == 0 {
			writeExecutorError(c, http.StatusInternalServerError, 5001, "compiled pipeline plan is empty")
			return
		}

		chain, err := e.compileHandlers(plan)
		if err != nil {
			writeExecutorError(c, http.StatusInternalServerError, 5002, err.Error())
			return
		}

		c.Set(CtxExecutorEnabled, true)
		if isDebugPlan(c) {
			c.Header("X-Pipeline-Executor", "1")
		}
		runHandlersChain(c, chain)
		if !c.IsAborted() {
			// Executor owns the proxy main chain and should stop outer gin handlers.
			c.Abort()
		}
	}
}

func (e *Executor) getOrBuildPlan(c *gin.Context, service *dao.ServiceDetail) (*Plan, error) {
	if plan, ok := GetPlan(c); ok && plan != nil {
		return plan, nil
	}
	plan, err := BuildPlanForService(c, service)
	if err != nil {
		return nil, err
	}
	if plan != nil {
		c.Set(CtxPlanKey, plan)
	}
	return plan, nil
}

func (e *Executor) compileHandlers(plan *Plan) (gin.HandlersChain, error) {
	if e == nil || e.registry == nil {
		return nil, fmt.Errorf("executor registry is nil")
	}
	out := make(gin.HandlersChain, 0, len(plan.Plugins))

	for _, pluginName := range plan.Plugins {
		plugin, ok := e.registry.Get(pluginName)
		if !ok || plugin == nil {
			return nil, fmt.Errorf("plugin not registered: %s", pluginName)
		}
		handler := e.toHandler(plugin, plan.ConfigVersion)
		out = append(out, handler)
	}
	return out, nil
}

func (e *Executor) toHandler(plugin http_proxy_plugin.Plugin, planVersion string) gin.HandlerFunc {
	if middlewarePlugin, ok := plugin.(MiddlewarePlugin); ok && middlewarePlugin.Handler() != nil {
		legacy := middlewarePlugin.Handler()
		return func(c *gin.Context) {
			ec := http_proxy_plugin.NewExecContext(c)
			ec.PlanVersion = planVersion
			if !plugin.Enabled(ec) {
				c.Next()
				return
			}
			legacy(c)
		}
	}

	return func(c *gin.Context) {
		ec := http_proxy_plugin.NewExecContext(c)
		ec.PlanVersion = planVersion

		if !plugin.Enabled(ec) {
			c.Next()
			return
		}

		result := plugin.Execute(ec)
		if result.IsAbort() {
			handlePluginAbort(c, result)
			return
		}
		c.Next()
	}
}

func handlePluginAbort(c *gin.Context, result http_proxy_plugin.Result) {
	if c.IsAborted() {
		return
	}
	status := result.HTTPStatus
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	code := result.Code
	if code <= 0 {
		code = 5000
	}
	message := result.Message
	if message == "" && result.Err != nil {
		message = result.Err.Error()
	}
	if message == "" {
		message = "plugin aborted request"
	}
	writeExecutorError(c, status, code, message)
}

func getServiceFromContext(c *gin.Context) (*dao.ServiceDetail, bool) {
	if c == nil {
		return nil, false
	}
	raw, ok := c.Get("service")
	if !ok {
		return nil, false
	}
	service, ok := raw.(*dao.ServiceDetail)
	if !ok || service == nil || service.Info == nil {
		return nil, false
	}
	return service, true
}

func writeExecutorError(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"errno":    code,
		"errmsg":   message,
		"data":     "",
		"trace_id": "",
	})
}
