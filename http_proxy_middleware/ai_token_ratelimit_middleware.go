package http_proxy_middleware

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

// AITokenRateLimitMiddleware enforces token-level rate limiting.
func AITokenRateLimitMiddleware() gin.HandlerFunc {
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

		body, err := aiReadBody(c)
		if err != nil {
			middleware.ResponseError(c, 3203, err)
			return
		}
		aiResetBody(c, body)

		estimatedTokens := aiEstimateTokens(body)

		limiter := ai_gateway.GetTokenLimiter()
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
