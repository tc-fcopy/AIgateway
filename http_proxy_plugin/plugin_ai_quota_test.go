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

type mockQuotaManager struct {
	quotaLeft   int64
	getErr      error
	consumeOK   bool
	consumeErr  error
	getCalls    int
	consumeCall int
	deltaCalls  int
	lastConsume struct {
		consumer string
		tokens   int64
	}
	lastDelta struct {
		consumer string
		delta    int64
	}
}

func (m *mockQuotaManager) GetQuota(_ *gin.Context, consumerName string) (int64, error) {
	m.getCalls++
	if m.getErr != nil {
		return 0, m.getErr
	}
	return m.quotaLeft, nil
}

func (m *mockQuotaManager) ConsumeQuota(_ *gin.Context, consumerName string, tokens int64) (bool, error) {
	m.consumeCall++
	m.lastConsume.consumer = consumerName
	m.lastConsume.tokens = tokens
	if m.consumeErr != nil {
		return false, m.consumeErr
	}
	return m.consumeOK, nil
}

func (m *mockQuotaManager) DeltaQuota(_ *gin.Context, consumerName string, delta int64) error {
	m.deltaCalls++
	m.lastDelta.consumer = consumerName
	m.lastDelta.delta = delta
	return nil
}

func TestAIQuotaPluginQuotaExceededBeforeConsume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIQuotaTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableQuota: true,
		},
	})

	mock := &mockQuotaManager{quotaLeft: 1, consumeOK: true}
	aiQuotaManagerGetter = func() quotaManagerLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "quota-consumer")
		c.Next()
	})
	r.Use(NewAIQuotaPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"1234567890"}`))
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "quota exceeded") {
		t.Fatalf("expected quota exceeded message, got %s", w.Body.String())
	}
	if mock.consumeCall != 0 {
		t.Fatalf("expected no consume call when quota insufficient")
	}
}

func TestAIQuotaPluginConsumeAndDelta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIQuotaTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableQuota: true,
		},
	})

	mock := &mockQuotaManager{quotaLeft: 1000, consumeOK: true}
	aiQuotaManagerGetter = func() quotaManagerLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "quota-consumer")
		c.Next()
	})
	r.Use(NewAIQuotaPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.Set(aigwctx.TokenUsageKey, &token.TokenUsage{TotalTokens: 100})
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"1234567890"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mock.getCalls != 1 || mock.consumeCall != 1 || mock.deltaCalls != 1 {
		t.Fatalf("expected get=1 consume=1 delta=1, got get=%d consume=%d delta=%d", mock.getCalls, mock.consumeCall, mock.deltaCalls)
	}
	expectedDelta := mock.lastConsume.tokens - 100
	if mock.lastDelta.delta != expectedDelta {
		t.Fatalf("expected delta %d, got %d", expectedDelta, mock.lastDelta.delta)
	}
}

func TestAIQuotaPluginGetQuotaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIQuotaTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableQuota: true,
		},
	})

	mock := &mockQuotaManager{getErr: errors.New("quota db error")}
	aiQuotaManagerGetter = func() quotaManagerLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "quota-consumer")
		c.Next()
	})
	r.Use(NewAIQuotaPlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"hello"}`))
	r.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "quota db error") {
		t.Fatalf("expected get quota error in response, got %s", w.Body.String())
	}
}

func prepareAIQuotaTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevGetter := aiQuotaManagerGetter
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiQuotaManagerGetter = prevGetter
	}
}
