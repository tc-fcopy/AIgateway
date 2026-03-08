package http_proxy_middleware

import (
	"fmt"
	"time"

	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/observability"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

// AIObservabilityMiddleware collects logs and metrics for AI requests.
func AIObservabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsObservabilityEnabled() {
			c.Next()
			return
		}

		start := time.Now()
		requestID := fmt.Sprintf("%d", start.UnixNano())
		observability.GlobalLogger.SetRequestID(requestID)

		observability.GlobalLogger.WithFields(map[string]interface{}{
			"request_id": requestID,
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"ip":         c.ClientIP(),
		}).Info("request started")

		c.Set(aigwctx.StartTimeKey, start.UnixNano())
		c.Set("request_id", requestID)
		c.Next()

		duration := time.Since(start)
		service := c.GetString("service_name")
		model := c.GetString(aigwctx.ModelKey)
		consumer := c.GetString(aigwctx.ConsumerNameKey)
		statusCode := c.Writer.Status()

		totalTokens := int64(0)
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok {
				totalTokens = tokenUsage.TotalTokens
			}
		}

		observability.GlobalMetrics.RecordRequest(service, model, statusCode, totalTokens)

		fields := map[string]interface{}{
			"request_id":  requestID,
			"service":     service,
			"model":       model,
			"consumer":    consumer,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"tokens":      totalTokens,
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
		}

		if statusCode >= 400 {
			observability.GlobalLogger.WithFields(fields).Warn("request completed with error")
		} else {
			observability.GlobalLogger.WithFields(fields).Info("request completed")
		}

		observability.GlobalLogger.ClearRequestID()
	}
}

func GetRequestID(c *gin.Context) (string, bool) {
	requestID, exists := c.Get("request_id")
	if !exists {
		return "", false
	}
	id, ok := requestID.(string)
	return id, ok
}
