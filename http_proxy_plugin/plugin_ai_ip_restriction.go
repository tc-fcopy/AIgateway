package http_proxy_plugin

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
)

// AIIPRestrictionPlugin is the native plugin migration for ai.ip_restriction.
type AIIPRestrictionPlugin struct{}

func NewAIIPRestrictionPlugin() *AIIPRestrictionPlugin {
	return &AIIPRestrictionPlugin{}
}

func (p *AIIPRestrictionPlugin) Name() string {
	return PluginAIIPRestriction
}

func (p *AIIPRestrictionPlugin) Phase() Phase {
	return PhasePolicy
}

func (p *AIIPRestrictionPlugin) Priority() int {
	return 1000
}

func (p *AIIPRestrictionPlugin) Requires() []string {
	return nil
}

func (p *AIIPRestrictionPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIIPRestrictionPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsIPRestrictionEnabled() {
		return Continue()
	}

	conf := config.AIConfManager.GetConfig()
	if conf == nil {
		return Continue()
	}

	manager := ai_gateway.GetIPRestrictionManager()
	if manager == nil {
		return Continue()
	}

	manager.SetGlobalRules(conf.IPRestriction.EnableCIDR, conf.IPRestriction.Whitelist, conf.IPRestriction.Blacklist)

	ip := ctx.Gin.ClientIP()
	consumerName := ctx.Gin.GetString(aigwctx.ConsumerNameKey)
	if !manager.IsAllowed(ip, consumerName) {
		err := errors.New("ip is not allowed")
		middleware.ResponseError(ctx.Gin, 3601, err)
		return AbortWithStatus(ctx.Gin.Writer.Status(), err)
	}

	return Continue()
}
