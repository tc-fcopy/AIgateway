package http_proxy_plugin

import (
	"errors"
	"strings"

	"gateway/dao"
	"gateway/middleware"
)

// ProxyHeaderTransferPlugin is the native plugin migration for proxy.header_transfer.
type ProxyHeaderTransferPlugin struct{}

func NewProxyHeaderTransferPlugin() *ProxyHeaderTransferPlugin {
	return &ProxyHeaderTransferPlugin{}
}

func (p *ProxyHeaderTransferPlugin) Name() string {
	return PluginProxyHeader
}

func (p *ProxyHeaderTransferPlugin) Phase() Phase {
	return PhaseTransform
}

func (p *ProxyHeaderTransferPlugin) Priority() int {
	return 500
}

func (p *ProxyHeaderTransferPlugin) Requires() []string {
	return nil
}

func (p *ProxyHeaderTransferPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *ProxyHeaderTransferPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	serverInterface, ok := ctx.Gin.Get("service")
	if !ok {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}

	serviceDetail, ok := serverInterface.(*dao.ServiceDetail)
	if !ok || serviceDetail == nil || serviceDetail.HTTPRule == nil {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}

	for _, item := range strings.Split(serviceDetail.HTTPRule.HeaderTransfor, ",") {
		items := strings.Split(item, " ")
		if len(items) != 3 {
			continue
		}
		if items[0] == "add" || items[0] == "edit" {
			ctx.Gin.Request.Header.Set(items[1], items[2])
		}
		if items[0] == "del" {
			ctx.Gin.Request.Header.Del(items[1])
		}
	}

	return Continue()
}
