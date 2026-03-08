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

func (p *AITokenRateLimitPlugin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsTokenRateLimitEnabled() {
			c.Next()
			return
		}

		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		if consumerName == "" {
			c.Next()
			return
		}

		serviceName := c.GetString("service_name")
		if serviceName == "" {
			serviceName = "default"
		}

		body, err := pluginReadBody(c)
		if err != nil {
			middleware.ResponseError(c, 3203, err)
			return
		}
		pluginResetBody(c, body)

		estimatedTokens := pluginEstimateTokens(body)

		limiter := aiTokenLimiterGetter()
		if limiter == nil {
			c.Next()
			return
		}

		allowed, err := limiter.CheckLimit(c, serviceName, consumerName, estimatedTokens)
		if err != nil {
			middleware.ResponseError(c, 3204, err)
			return
		}
		if !allowed {
			middleware.ResponseError(c, 5002, errors.New("token rate limit exceeded"))
			return
		}

		c.Next()

		if c.IsAborted() {
			return
		}

		actualTokens := estimatedTokens
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok && tokenUsage.TotalTokens > 0 {
				actualTokens = tokenUsage.TotalTokens
			}
		}

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
