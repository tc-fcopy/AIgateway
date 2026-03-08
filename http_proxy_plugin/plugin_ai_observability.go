package http_proxy_plugin

import (
	"errors"
	"fmt"
	"time"

	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/observability"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

type observabilityEntryLike interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

type observabilityLoggerLike interface {
	SetRequestID(requestID string)
	WithFields(fields map[string]interface{}) observabilityEntryLike
	ClearRequestID()
}

type observabilityMetricsLike interface {
	RecordRequest(service, model string, statusCode int, totalTokens int64)
}

type loggerAdapter struct{}

func (loggerAdapter) SetRequestID(requestID string) {
	observability.GlobalLogger.SetRequestID(requestID)
}

func (loggerAdapter) WithFields(fields map[string]interface{}) observabilityEntryLike {
	return observability.GlobalLogger.WithFields(fields)
}

func (loggerAdapter) ClearRequestID() {
	observability.GlobalLogger.ClearRequestID()
}

var aiObservabilityLoggerGetter = func() observabilityLoggerLike {
	return loggerAdapter{}
}

var aiObservabilityMetricsGetter = func() observabilityMetricsLike {
	return observability.GlobalMetrics
}

// AIObservabilityPlugin is the native plugin migration for ai.observability.
// It keeps request start/end logging and metrics recording semantics.
type AIObservabilityPlugin struct{}

func NewAIObservabilityPlugin() *AIObservabilityPlugin {
	return &AIObservabilityPlugin{}
}

func (p *AIObservabilityPlugin) Name() string {
	return PluginAIObservability
}

func (p *AIObservabilityPlugin) Phase() Phase {
	return PhaseObserve
}

func (p *AIObservabilityPlugin) Priority() int {
	return 1000
}

func (p *AIObservabilityPlugin) Requires() []string {
	return nil
}

func (p *AIObservabilityPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIObservabilityPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}
	return Continue()
}

func (p *AIObservabilityPlugin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsObservabilityEnabled() {
			c.Next()
			return
		}

		logger := aiObservabilityLoggerGetter()
		metrics := aiObservabilityMetricsGetter()
		if logger == nil || metrics == nil {
			c.Next()
			return
		}

		start := time.Now()
		requestID := fmt.Sprintf("%d", start.UnixNano())
		logger.SetRequestID(requestID)
		logger.WithFields(map[string]interface{}{
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
		modelName := c.GetString(aigwctx.ModelKey)
		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		statusCode := c.Writer.Status()

		totalTokens := int64(0)
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok {
				totalTokens = tokenUsage.TotalTokens
			}
		}

		metrics.RecordRequest(service, modelName, statusCode, totalTokens)

		fields := map[string]interface{}{
			"request_id":  requestID,
			"service":     service,
			"model":       modelName,
			"consumer":    consumerName,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"tokens":      totalTokens,
			"path":        c.Request.URL.Path,
			"method":      c.Request.Method,
		}

		if statusCode >= 400 {
			logger.WithFields(fields).Warn("request completed with error")
		} else {
			logger.WithFields(fields).Info("request completed")
		}
		logger.ClearRequestID()
	}
}
