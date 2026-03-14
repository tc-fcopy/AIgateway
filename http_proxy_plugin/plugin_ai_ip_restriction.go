package http_proxy_plugin

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
)

// AIIPRestrictionPlugin AI网关 IP 访问限制插件
// 功能：实现全局 IP 白名单、黑名单、CIDR 网段访问控制，拦截非法 IP 请求
type AIIPRestrictionPlugin struct{}

// NewAIIPRestrictionPlugin 创建 IP 限制插件实例
func NewAIIPRestrictionPlugin() *AIIPRestrictionPlugin {
	return &AIIPRestrictionPlugin{}
}

// Name 返回插件唯一名称
func (p *AIIPRestrictionPlugin) Name() string {
	return PluginAIIPRestriction
}

// Phase 插件执行阶段：Policy 策略阶段（认证后、转发前，做安全校验）
func (p *AIIPRestrictionPlugin) Phase() Phase {
	return PhasePolicy
}

// Priority 执行优先级：1000（同阶段最高，最先执行）
func (p *AIIPRestrictionPlugin) Priority() int {
	return 1000
}

// Requires 插件依赖：无
func (p *AIIPRestrictionPlugin) Requires() []string {
	return nil
}

// Enabled 插件是否启用：默认始终启用
func (p *AIIPRestrictionPlugin) Enabled(*ExecContext) bool {
	return true
}

// Execute 插件核心执行逻辑（IP 校验主流程）
func (p *AIIPRestrictionPlugin) Execute(ctx *ExecContext) Result {
	// 1. 上下文合法性校验
	if ctx == nil || ctx.Gin == nil {
		// 上下文为空，中断请求
		return Abort(errors.New("execution context is nil"))
	}

	// 2. 功能开关校验：网关未启用 / IP 限制未开启，直接跳过插件
	if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsIPRestrictionEnabled() {
		return Continue()
	}

	// 3. 获取网关配置，配置为空则跳过
	conf := config.AIConfManager.GetConfig()
	if conf == nil {
		return Continue()
	}

	// 4. 获取 IP 限制管理器，实例为空则跳过
	manager := ai_gateway.GetIPRestrictionManager()
	if manager == nil {
		return Continue()
	}

	// 5. 加载全局 IP 规则：CIDR开关、白名单、黑名单
	manager.SetGlobalRules(
		conf.IPRestriction.EnableCIDR,
		conf.IPRestriction.Whitelist,
		conf.IPRestriction.Blacklist,
	)

	// 6. 获取客户端真实 IP
	ip := ctx.Gin.ClientIP()
	// 7. 获取当前请求的用户标识（来自认证插件）
	consumerName := ctx.Gin.GetString(aigwctx.ConsumerNameKey)

	// 8. 核心校验：判断当前 IP 是否允许访问
	if !manager.IsAllowed(ip, consumerName) {
		// IP 不合法，构造错误信息
		err := errors.New("ip is not allowed")
		// 返回固定错误码 3601：IP 禁止访问
		middleware.ResponseError(ctx.Gin, 3601, err)
		// 中断请求，不再执行后续插件
		return AbortWithStatus(ctx.Gin.Writer.Status(), err)
	}

	// 9. IP 校验通过，继续执行后续插件
	return Continue()
}
