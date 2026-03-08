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

type quotaManagerLike interface {
	GetQuota(c *gin.Context, consumerName string) (int64, error)
	ConsumeQuota(c *gin.Context, consumerName string, tokens int64) (bool, error)
	DeltaQuota(c *gin.Context, consumerName string, delta int64) error
}

var aiQuotaManagerGetter = func() quotaManagerLike {
	return ai_gateway.GetQuotaManager()
}

// AIQuotaPlugin is the native plugin migration for ai.quota.
// It keeps request-phase consume + response-phase delta correction semantics.
type AIQuotaPlugin struct{}

func NewAIQuotaPlugin() *AIQuotaPlugin {
	return &AIQuotaPlugin{}
}

func (p *AIQuotaPlugin) Name() string {
	return PluginAIQuota
}

func (p *AIQuotaPlugin) Phase() Phase {
	return PhasePolicy
}

func (p *AIQuotaPlugin) Priority() int {
	return 800
}

func (p *AIQuotaPlugin) Requires() []string {
	return []string{PluginAIAuth}
}

func (p *AIQuotaPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIQuotaPlugin) Execute(*ExecContext) Result {
	return Continue()
}

func (p *AIQuotaPlugin) Handler() gin.HandlerFunc {
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

		body, err := pluginQuotaReadBody(c)
		if err != nil {
			middleware.ResponseError(c, 3301, err)
			return
		}
		pluginQuotaResetBody(c, body)
		estimatedTokens := pluginQuotaEstimateTokens(body)

		manager := aiQuotaManagerGetter()
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

		_ = manager.DeltaQuota(c, consumerName, estimatedTokens-actual)
	}
}

func pluginQuotaReadBody(c *gin.Context) ([]byte, error) {
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

func pluginQuotaResetBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
}

func pluginQuotaEstimateTokens(body []byte) int64 {
	if len(body) == 0 {
		return 1
	}
	n := int64(len(bytes.TrimSpace(body))) / 4
	if n <= 0 {
		return 1
	}
	return n
}
