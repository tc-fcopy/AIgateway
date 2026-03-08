package http_proxy_plugin

import (
	"errors"

	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
)

// CoreFlowCountPlugin is the first native plugin migrated from legacy middleware.
type CoreFlowCountPlugin struct{}

func NewCoreFlowCountPlugin() *CoreFlowCountPlugin {
	return &CoreFlowCountPlugin{}
}

func (p *CoreFlowCountPlugin) Name() string {
	return PluginCoreFlowCount
}

func (p *CoreFlowCountPlugin) Phase() Phase {
	return PhaseEdgeGuard
}

func (p *CoreFlowCountPlugin) Priority() int {
	return 900
}

func (p *CoreFlowCountPlugin) Requires() []string {
	return nil
}

func (p *CoreFlowCountPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *CoreFlowCountPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	serverInterface, ok := ctx.Gin.Get("service")
	if !ok {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}

	serviceDetail, ok := serverInterface.(*dao.ServiceDetail)
	if !ok || serviceDetail == nil || serviceDetail.Info == nil {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}

	totalCounter, err := public.FlowCounterHandler.GetCounter(public.FlowTotal)
	if err != nil {
		middleware.ResponseError(ctx.Gin, 4001, err)
		return AbortWithStatus(ctx.Gin.Writer.Status(), err)
	}
	totalCounter.Increase()

	serviceCounter, err := public.FlowCounterHandler.GetCounter(public.FlowServicePrefix + serviceDetail.Info.ServiceName)
	if err != nil {
		middleware.ResponseError(ctx.Gin, 4001, err)
		return AbortWithStatus(ctx.Gin.Writer.Status(), err)
	}
	serviceCounter.Increase()

	return Continue()
}
