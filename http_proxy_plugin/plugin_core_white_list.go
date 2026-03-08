package http_proxy_plugin

import (
	"errors"
	"fmt"
	"strings"

	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
)

// CoreWhiteListPlugin is the native plugin migration for core.white_list.
type CoreWhiteListPlugin struct{}

func NewCoreWhiteListPlugin() *CoreWhiteListPlugin {
	return &CoreWhiteListPlugin{}
}

func (p *CoreWhiteListPlugin) Name() string {
	return PluginCoreWhiteList
}

func (p *CoreWhiteListPlugin) Phase() Phase {
	return PhaseEdgeGuard
}

func (p *CoreWhiteListPlugin) Priority() int {
	return 700
}

func (p *CoreWhiteListPlugin) Requires() []string {
	return nil
}

func (p *CoreWhiteListPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *CoreWhiteListPlugin) Execute(ctx *ExecContext) Result {
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

	iplist := []string{}
	if serviceDetail.AccessControl.WhiteList != "" {
		iplist = strings.Split(serviceDetail.AccessControl.WhiteList, ",")
	}

	if serviceDetail.AccessControl.OpenAuth == 1 && len(iplist) > 0 {
		if !public.InStringSlice(iplist, ctx.Gin.ClientIP()) {
			middleware.ResponseError(ctx.Gin, 3001, errors.New(fmt.Sprintf("%s not in white ip list", ctx.Gin.ClientIP())))
			return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
		}
	}

	return Continue()
}
