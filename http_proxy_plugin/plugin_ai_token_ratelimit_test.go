package http_proxy_plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

type mockTokenLimiter struct {
	allow      bool
	checkErr   error
	checkCalls int
	lastCheck  struct {
		service  string
		consumer string
		tokens   int64
	}
	updateCalls int
	lastUpdate  struct {
		service  string
		consumer string
		tokens   int64
	}
}

func (m *mockTokenLimiter) CheckLimit(_ *gin.Context, serviceName, consumerName string, estimatedTokens int64) (bool, error) {
	m.checkCalls++
	m.lastCheck.service = serviceName
	m.lastCheck.consumer = consumerName
	m.lastCheck.tokens = estimatedTokens
	if m.checkErr != nil {
		return false, m.checkErr
	}
	return m.allow, nil
}

func (m *mockTokenLimiter) UpdateCount(_ *gin.Context, serviceName, consumerName string, actualTokens int64) error {
	m.updateCalls++
	m.lastUpdate.service = serviceName
	m.lastUpdate.consumer = consumerName
	m.lastUpdate.tokens = actualTokens
	return nil
}

func TestAITokenRateLimitPluginLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAITokenLimitTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableTokenRateLimit: true,
		},
	})

	mock := &mockTokenLimiter{allow: false}
	aiTokenLimiterGetter = func() tokenLimiterLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "cons-a")
		c.Set("service_name", "svc-a")
		c.Next()
	})
	r.Use(NewAITokenRateLimitPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"hello world"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when limited, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "token rate limit exceeded") {
		t.Fatalf("expected rate limit message, got %s", w.Body.String())
	}
	if mock.updateCalls != 0 {
		t.Fatalf("expected no update call on limited request")
	}
}

func TestAITokenRateLimitPluginUpdateWithActualTokenUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAITokenLimitTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableTokenRateLimit: true,
		},
	})

	mock := &mockTokenLimiter{allow: true}
	aiTokenLimiterGetter = func() tokenLimiterLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "cons-b")
		c.Set("service_name", "svc-b")
		c.Next()
	})
	r.Use(NewAITokenRateLimitPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.Set(aigwctx.TokenUsageKey, &token.TokenUsage{TotalTokens: 123})
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"hello world"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.checkCalls != 1 || mock.updateCalls != 1 {
		t.Fatalf("expected check=1 update=1, got check=%d update=%d", mock.checkCalls, mock.updateCalls)
	}
	if mock.lastUpdate.tokens != 123 {
		t.Fatalf("expected update with actual tokens 123, got %d", mock.lastUpdate.tokens)
	}
}

func TestAITokenRateLimitPluginCheckError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAITokenLimitTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableTokenRateLimit: true,
		},
	})

	mock := &mockTokenLimiter{checkErr: errors.New("redis down")}
	aiTokenLimiterGetter = func() tokenLimiterLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "cons-c")
		c.Set("service_name", "svc-c")
		c.Next()
	})
	r.Use(NewAITokenRateLimitPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"hello world"}`))
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "redis down") {
		t.Fatalf("expected check error in response, got %s", w.Body.String())
	}
}

func prepareAITokenLimitTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevGetter := aiTokenLimiterGetter
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiTokenLimiterGetter = prevGetter
	}
}
