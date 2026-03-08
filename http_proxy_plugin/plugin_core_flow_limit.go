package http_proxy_plugin

import (
	"errors"
	"fmt"

	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
)

// CoreFlowLimitPlugin is the native plugin migration for core.flow_limit.
type CoreFlowLimitPlugin struct{}

func NewCoreFlowLimitPlugin() *CoreFlowLimitPlugin {
	return &CoreFlowLimitPlugin{}
}

func (p *CoreFlowLimitPlugin) Name() string {
	return PluginCoreFlowLimit
}

func (p *CoreFlowLimitPlugin) Phase() Phase {
	return PhaseEdgeGuard
}

func (p *CoreFlowLimitPlugin) Priority() int {
	return 800
}

func (p *CoreFlowLimitPlugin) Requires() []string {
	return nil
}

func (p *CoreFlowLimitPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *CoreFlowLimitPlugin) Execute(ctx *ExecContext) Result {
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

	accessControl := serviceDetail.AccessControl
	if accessControl == nil {
		return Continue()
	}

	if accessControl.ServiceFlowLimit != 0 {
		serviceLimiter, err := public.FlowLimiterHandler.GetLimiter(
			public.FlowServicePrefix+serviceDetail.Info.ServiceName,
			float64(accessControl.ServiceFlowLimit),
		)
		if err != nil {
			middleware.ResponseError(ctx.Gin, 5001, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}
		if !serviceLimiter.Allow() {
			middleware.ResponseError(
				ctx.Gin,
				5002,
				errors.New(fmt.Sprintf("service flow limit %v", accessControl.ServiceFlowLimit)),
			)
			return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
		}
	}

	if accessControl.ClientIPFlowLimit > 0 {
		clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
			public.FlowServicePrefix+serviceDetail.Info.ServiceName+"_"+ctx.Gin.ClientIP(),
			float64(accessControl.ClientIPFlowLimit),
		)
		if err != nil {
			middleware.ResponseError(ctx.Gin, 5003, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}
		if !clientLimiter.Allow() {
			middleware.ResponseError(
				ctx.Gin,
				5002,
				errors.New(fmt.Sprintf("%v flow limit %v", ctx.Gin.ClientIP(), accessControl.ClientIPFlowLimit)),
			)
			return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
		}
	}

	return Continue()
}
