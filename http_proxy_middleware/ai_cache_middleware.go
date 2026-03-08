package http_proxy_middleware

import (
	"net/http"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

// AICacheMiddleware caches AI responses by consumer+model+prompt key.
func AICacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsCacheEnabled() {
			c.Next()
			return
		}

		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		if consumerName == "" {
			c.Next()
			return
		}

		cacheStore := ai_gateway.GetStringCache()
		if cacheStore == nil {
			c.Next()
			return
		}

		body, err := aiReadBody(c)
		if err != nil {
			c.Next()
			return
		}
		aiResetBody(c, body)

		payload, _ := aiParseJSONBody(body)
		model := aiGetModel(payload, c)
		if model == "" {
			model = "default"
		}

		cacheKey, err := cacheStore.GenerateCacheKeyFromRequest(consumerName, model, body)
		if err != nil {
			c.Next()
			return
		}

		c.Set(aigwctx.CacheKey, cacheKey)

		if entry, err := cacheStore.Get(cacheKey); err == nil && entry != nil {
			c.Header("X-AI-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", entry.Response)
			c.Abort()
			return
		}

		writer := ai_gateway.GetStringCache()
		_ = writer

		cacheWriter := tokenAwareCacheWriter{ResponseWriter: c.Writer}
		c.Writer = &cacheWriter
		c.Next()

		if c.IsAborted() {
			return
		}

		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}

		responseBody := cacheWriter.body
		if len(responseBody) == 0 {
			return
		}

		tokenCount := int64(0)
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok {
				tokenCount = tokenUsage.TotalTokens
			}
		}

		_ = cacheStore.Set(cacheKey, responseBody, int(tokenCount))
		c.Header("X-AI-Cache", "MISS")
	}
}

type tokenAwareCacheWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *tokenAwareCacheWriter) Write(data []byte) (int, error) {
	if len(data) > 0 {
		w.body = append(w.body, data...)
	}
	return w.ResponseWriter.Write(data)
}
