package http_proxy_plugin

import (
	"net/http"

	"gateway/ai_gateway"
	aicache "gateway/ai_gateway/cache"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

type stringCacheLike interface {
	GenerateCacheKeyFromRequest(consumer, model string, body []byte) (string, error)
	Get(cacheKey string) (*aicache.CacheEntry, error)
	Set(cacheKey string, response []byte, tokenCount int) error
}

var aiStringCacheGetter = func() stringCacheLike {
	return ai_gateway.GetStringCache()
}

// AICachePlugin is the native plugin migration for ai.cache.
// It keeps request-phase HIT short-circuit and response-phase MISS writeback semantics.
type AICachePlugin struct{}

func NewAICachePlugin() *AICachePlugin {
	return &AICachePlugin{}
}

func (p *AICachePlugin) Name() string {
	return PluginAICache
}

func (p *AICachePlugin) Phase() Phase {
	return PhaseTraffic
}

func (p *AICachePlugin) Priority() int {
	return 1000
}

func (p *AICachePlugin) Requires() []string {
	return []string{PluginAIAuth}
}

func (p *AICachePlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AICachePlugin) Execute(*ExecContext) Result {
	return Continue()
}

func (p *AICachePlugin) Handler() gin.HandlerFunc {
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

		cacheStore := aiStringCacheGetter()
		if cacheStore == nil {
			c.Next()
			return
		}

		body, err := pluginReadBody(c)
		if err != nil {
			c.Next()
			return
		}
		pluginResetBody(c, body)

		payload, _ := pluginParseJSONBody(body)
		ec := NewExecContext(c)
		modelName := pluginGetModel(payload, ec)
		ReleaseExecContext(ec)
		if modelName == "" {
			modelName = "default"
		}

		cacheKey, err := cacheStore.GenerateCacheKeyFromRequest(consumerName, modelName, body)
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

		cacheWriter := pluginCacheWriter{ResponseWriter: c.Writer}
		c.Writer = &cacheWriter
		c.Next()

		if c.IsAborted() {
			return
		}
		if c.Writer.Status() < 200 || c.Writer.Status() >= 300 {
			return
		}
		if len(cacheWriter.body) == 0 {
			return
		}

		tokenCount := int64(0)
		if usage, ok := c.Get(aigwctx.TokenUsageKey); ok {
			if tokenUsage, ok := usage.(*token.TokenUsage); ok {
				tokenCount = tokenUsage.TotalTokens
			}
		}

		_ = cacheStore.Set(cacheKey, cacheWriter.body, int(tokenCount))
		c.Header("X-AI-Cache", "MISS")
	}
}

type pluginCacheWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *pluginCacheWriter) Write(data []byte) (int, error) {
	if len(data) > 0 {
		w.body = append(w.body, data...)
	}
	return w.ResponseWriter.Write(data)
}
