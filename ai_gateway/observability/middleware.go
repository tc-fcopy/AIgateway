package observability

import (
	"github.com/gin-gonic/gin"
)

// ObservabilityMiddleware records request-level metrics.
func ObservabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		service := c.GetString("service_name")
		model := c.GetString("ai_model")
		GlobalMetrics.RecordRequest(service, model, c.Writer.Status(), 0)
	}
}
