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

// AIQuotaMiddleware enforces per-consumer quota.
func AIQuotaMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsQuotaEnabled() {
			c.Next()
			return
		}

		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		if consumerName == "" {
			c.Next()
			return
		}

		body, err := aiReadBody(c)
		if err != nil {
			middleware.ResponseError(c, 3301, err)
			return
		}
		aiResetBody(c, body)
		estimatedTokens := aiEstimateTokens(body)

		manager := ai_gateway.GetQuotaManager()
		if manager == nil {
			c.Next()
			return
		}

		quotaLeft, err := manager.GetQuota(c, consumerName)
		if err != nil {
			middleware.ResponseError(c, 3302, err)
			return
		}
		if quotaLeft < estimatedTokens {
			middleware.ResponseError(c, 3303, errors.New("quota exceeded"))
			return
		}

		ok, err := manager.ConsumeQuota(c, consumerName, estimatedTokens)
		if err != nil {
			middleware.ResponseError(c, 3304, err)
			return
		}
		if !ok {
			middleware.ResponseError(c, 3305, errors.New("quota exceeded"))
			return
		}

		c.Next()

		if c.IsAborted() {
			return
		}

		actual := estimatedTokens
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok && tokenUsage.TotalTokens > 0 {
				actual = tokenUsage.TotalTokens
			}
		}

		// refund or extra deduction delta.
		_ = manager.DeltaQuota(c, consumerName, estimatedTokens-actual)
	}
}
