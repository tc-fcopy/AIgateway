package http_proxy_plugin

import (
	"bytes"
	"errors"
	"io"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

type tokenLimiterLike interface {
	CheckLimit(c *gin.Context, serviceName, consumerName string, estimatedTokens int64) (bool, error)
	UpdateCount(c *gin.Context, serviceName, consumerName string, actualTokens int64) error
}

var aiTokenLimiterGetter = func() tokenLimiterLike {
	return ai_gateway.GetTokenLimiter()
}

// AITokenRateLimitPlugin is the native plugin migration for ai.token_ratelimit.
// It keeps request-phase check + response-phase update semantics.
type AITokenRateLimitPlugin struct{}

func NewAITokenRateLimitPlugin() *AITokenRateLimitPlugin {
	return &AITokenRateLimitPlugin{}
}

func (p *AITokenRateLimitPlugin) Name() string {
	return PluginAITokenRateLimit
}

func (p *AITokenRateLimitPlugin) Phase() Phase {
	return PhasePolicy
}

func (p *AITokenRateLimitPlugin) Priority() int {
	return 900
}

func (p *AITokenRateLimitPlugin) Requires() []string {
	return []string{PluginAIAuth}
}

func (p *AITokenRateLimitPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AITokenRateLimitPlugin) Execute(*ExecContext) Result {
	return Continue()
}

// Handler 返回 Gin 中间件函数，是插件的核心执行逻辑
func (p *AITokenRateLimitPlugin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ====================== 步骤1：配置校验 ======================
		// 如果网关未启用 或 未开启Token限流功能，直接跳过该插件
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsTokenRateLimitEnabled() {
			c.Next() // 执行下一个中间件
			return   // 退出当前插件逻辑
		}

		// ====================== 步骤2：获取用户身份 ======================
		// 从上下文获取消费者名称（来自AIAuth认证插件，用户唯一标识）
		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		if consumerName == "" {
			c.Next()
			return
		}

		// ====================== 步骤3：获取服务名称 ======================
		// 从上下文获取AI服务名称
		serviceName := c.GetString("service_name")
		// 服务名称为空，使用默认值 default
		if serviceName == "" {
			serviceName = "default"
		}

		// ====================== 步骤4：读取请求体 ======================
		// 读取请求Body内容（Gin中Body只能读一次，必须读取后重置）
		body, err := pluginReadBody(c)
		if err != nil {
			// 读取Body失败，返回3203错误
			middleware.ResponseError(c, 3203, err)
			return
		}
		// 重置请求Body，保证后续插件/转发能正常读取Body
		pluginResetBody(c, body)

		// ====================== 步骤5：预估Token消耗 ======================
		// 根据请求体长度，粗略预估本次请求要消耗的Token数量
		estimatedTokens := pluginEstimateTokens(body)

		// ====================== 步骤6：获取Token限流器实例 ======================
		limiter := aiTokenLimiterGetter()
		// 限流器不存在，跳过限流校验
		if limiter == nil {
			c.Next()
			return
		}

		// ====================== 步骤7：执行Token限流校验 ======================
		// 检查：当前用户 + 当前服务 + 预估Token 是否超出限额
		allowed, err := limiter.CheckLimit(c, serviceName, consumerName, estimatedTokens)
		if err != nil {
			// 限流校验出错，返回3204错误
			middleware.ResponseError(c, 3204, err)
			return
		}
		// 如果超出限额，直接拦截请求，返回5002错误
		if !allowed {
			middleware.ResponseError(c, 5002, errors.New("token rate limit exceeded"))
			return
		}

		// ====================== 步骤8：放行请求，执行后续插件 ======================
		c.Next()

		// ====================== 步骤9：请求中断判断 ======================
		// 如果请求在后续流程中被中断（如报错、503），不更新Token计数
		if c.IsAborted() {
			return
		}

		// ====================== 步骤10：获取真实Token消耗量 ======================
		// 默认使用预估值
		actualTokens := estimatedTokens
		// 从上下文获取AI服务返回的真实Token使用量
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			// 类型断言成功，且真实Token>0，使用真实值
			if tokenUsage, ok := usage.(*token.TokenUsage); ok && tokenUsage.TotalTokens > 0 {
				actualTokens = tokenUsage.TotalTokens
			}
		}

		// ====================== 步骤11：更新限流计数器（真实结算） ======================
		// 用真实Token消耗量更新限流统计
		_ = limiter.UpdateCount(c, serviceName, consumerName, actualTokens)
	}
}
func pluginReadBody(c *gin.Context) ([]byte, error) {
	if c.Request == nil || c.Request.Body == nil {
		return []byte{}, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func pluginResetBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
}

func pluginEstimateTokens(body []byte) int64 {
	if len(body) == 0 {
		return 1
	}
	n := int64(len(bytes.TrimSpace(body))) / 4
	if n <= 0 {
		return 1
	}
	return n
}
