package http_proxy_plugin

import (
	"errors"
	"strings"

	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
)

// ProxyStripURIPlugin is the native plugin migration for proxy.strip_uri.
type ProxyStripURIPlugin struct{}

func NewProxyStripURIPlugin() *ProxyStripURIPlugin {
	return &ProxyStripURIPlugin{}
}

func (p *ProxyStripURIPlugin) Name() string {
	return PluginProxyStripURI
}

func (p *ProxyStripURIPlugin) Phase() Phase {
	return PhaseTransform
}

func (p *ProxyStripURIPlugin) Priority() int {
	return 400
}

func (p *ProxyStripURIPlugin) Requires() []string {
	return nil
}

func (p *ProxyStripURIPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *ProxyStripURIPlugin) Execute(ctx *ExecContext) Result {
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

	if serviceDetail.HTTPRule.RuleType == public.HTTPRuleTypePrefixURL && serviceDetail.HTTPRule.NeedStripUri == 1 {
		ctx.Gin.Request.URL.Path = strings.Replace(ctx.Gin.Request.URL.Path, serviceDetail.HTTPRule.Rule, "", 1)
	}

	return Continue()
}
