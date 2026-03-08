package http_proxy_plugin

import (
	"errors"
	"regexp"
	"strings"

	"gateway/dao"
	"gateway/middleware"
)

// ProxyURLRewritePlugin is the native plugin migration for proxy.url_rewrite.
type ProxyURLRewritePlugin struct{}

func NewProxyURLRewritePlugin() *ProxyURLRewritePlugin {
	return &ProxyURLRewritePlugin{}
}

func (p *ProxyURLRewritePlugin) Name() string {
	return PluginProxyURLRewrite
}

func (p *ProxyURLRewritePlugin) Phase() Phase {
	return PhaseTransform
}

func (p *ProxyURLRewritePlugin) Priority() int {
	return 300
}

func (p *ProxyURLRewritePlugin) Requires() []string {
	return nil
}

func (p *ProxyURLRewritePlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *ProxyURLRewritePlugin) Execute(ctx *ExecContext) Result {
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

	for _, item := range strings.Split(serviceDetail.HTTPRule.UrlRewrite, ",") {
		items := strings.Split(item, " ")
		if len(items) != 2 {
			continue
		}

		pattern, err := regexp.Compile(items[0])
		if err != nil {
			continue
		}

		replacePath := pattern.ReplaceAll([]byte(ctx.Gin.Request.URL.Path), []byte(items[1]))
		ctx.Gin.Request.URL.Path = string(replacePath)
	}

	return Continue()
}
