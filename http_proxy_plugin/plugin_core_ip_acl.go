package http_proxy_plugin

import (
	"errors"

	"gateway/dao"
	"gateway/middleware"
)

// CoreIPACLPlugin enforces service-level IP ACL with allow override + deny bloom.
type CoreIPACLPlugin struct{}

func NewCoreIPACLPlugin() *CoreIPACLPlugin {
	return &CoreIPACLPlugin{}
}

func (p *CoreIPACLPlugin) Name() string {
	return PluginCoreIPACL
}

func (p *CoreIPACLPlugin) Phase() Phase {
	return PhaseEdgeGuard
}

func (p *CoreIPACLPlugin) Priority() int {
	return 700
}

func (p *CoreIPACLPlugin) Requires() []string {
	return nil
}

func (p *CoreIPACLPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *CoreIPACLPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	serviceRaw, ok := ctx.Gin.Get("service")
	if !ok {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}
	serviceDetail, ok := serviceRaw.(*dao.ServiceDetail)
	if !ok || serviceDetail == nil || serviceDetail.Info == nil {
		middleware.ResponseError(ctx.Gin, 2001, errors.New("service not found"))
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}
	if serviceDetail.AccessControl == nil {
		return Continue()
	}

	access := serviceDetail.AccessControl
	if access.OpenAuth != 1 {
		return Continue()
	}

	acl := GetCoreIPACLManager().GetOrBuild(serviceDetail.Info.ID, access.OpenAuth, access.WhiteList, access.BlackList)
	if acl == nil {
		return Continue()
	}

	ip := ctx.Gin.ClientIP()
	if !acl.IsAllowed(ip) {
		err := errors.New("ip is not allowed")
		middleware.ResponseError(ctx.Gin, 3001, err)
		return AbortWithStatus(ctx.Gin.Writer.Status(), nil)
	}

	return Continue()
}
