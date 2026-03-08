package http_proxy_plugin

import (
	"errors"
	"fmt"
	"strings"

	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
)

// CoreBlackListPlugin is the native plugin migration for core.black_list.
type CoreBlackListPlugin struct{}

func NewCoreBlackListPlugin() *CoreBlackListPlugin {
	return &CoreBlackListPlugin{}
}

func (p *CoreBlackListPlugin) Name() string {
	return PluginCoreBlackList
}

func (p *CoreBlackListPlugin) Phase() Phase {
	return PhaseEdgeGuard
}

func (p *CoreBlackListPlugin) Priority() int {
	return 600
}

func (p *CoreBlackListPlugin) Requires() []string {
	return nil
}

func (p *CoreBlackListPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *CoreBlackListPlugin) Execute(ctx *ExecContext) Result {
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
	if serviceDetail.AccessControl == nil {
		return Continue()
	}

	whiteIPList := []string{}
	if serviceDetail.AccessControl.WhiteList != "" {
		whiteIPList = strings.Split(serviceDetail.AccessControl.WhiteList, ",")
	}

	blackIPList := []string{}
	if serviceDetail.AccessControl.BlackList != "" {
		blackIPList = strings.Split(serviceDetail.AccessControl.BlackList, ",")
	}

	if serviceDetail.AccessControl.OpenAuth == 1 && len(whiteIPList) == 0 && len(blackIPList) > 0 {
		if public.InStringSlice(blackIPList, ctx.Gin.ClientIP()) {
			middleware.ResponseError(ctx.Gin, 3001, errors.New(fmt.Sprintf("%s in black ip list", ctx.Gin.ClientIP())))
			return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
		}
	}

	return Continue()
}
