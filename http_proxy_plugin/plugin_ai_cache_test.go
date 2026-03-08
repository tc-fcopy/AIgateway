package http_proxy_plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aicache "gateway/ai_gateway/cache"
	aiconfig "gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

type mockStringCache struct {
	generateErr error
	getErr      error
	getEntry    *aicache.CacheEntry

	lastGenerate struct {
		consumer string
		model    string
		body     string
	}
	lastGetKey string

	setCalls int
	lastSet  struct {
		key   string
		body  []byte
		token int
	}
}

func (m *mockStringCache) GenerateCacheKeyFromRequest(consumer, model string, body []byte) (string, error) {
	m.lastGenerate.consumer = consumer
	m.lastGenerate.model = model
	m.lastGenerate.body = string(body)
	if m.generateErr != nil {
		return "", m.generateErr
	}
	return "cache-key-1", nil
}

func (m *mockStringCache) Get(cacheKey string) (*aicache.CacheEntry, error) {
	m.lastGetKey = cacheKey
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getEntry, nil
}

func (m *mockStringCache) Set(cacheKey string, response []byte, tokenCount int) error {
	m.setCalls++
	m.lastSet.key = cacheKey
	m.lastSet.body = append([]byte(nil), response...)
	m.lastSet.token = tokenCount
	return nil
}

func TestAICachePluginCacheHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAICachePluginTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableCache: true,
		},
	})

	mock := &mockStringCache{
		getEntry: &aicache.CacheEntry{Response: []byte(`{"cached":true}`)},
	}
	aiStringCacheGetter = func() stringCacheLike { return mock }

	var originCalled bool
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "consumer-a")
		c.Next()
	})
	r.Use(NewAICachePlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		originCalled = true
		c.Data(http.StatusOK, "application/json", []byte(`{"origin":true}`))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"model":"m1","prompt":"hi"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if originCalled {
		t.Fatalf("expected upstream handler skipped on cache hit")
	}
	if got := w.Header().Get("X-AI-Cache"); got != "HIT" {
		t.Fatalf("expected X-AI-Cache HIT, got %s", got)
	}
	if strings.TrimSpace(w.Body.String()) != `{"cached":true}` {
		t.Fatalf("unexpected hit body: %s", w.Body.String())
	}
	if mock.setCalls != 0 {
		t.Fatalf("expected no cache set on hit")
	}
}

func TestAICachePluginCacheMissAndWriteBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAICachePluginTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableCache: true,
		},
	})

	mock := &mockStringCache{
		getErr: errors.New("cache miss"),
	}
	aiStringCacheGetter = func() stringCacheLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "consumer-b")
		c.Next()
	})
	r.Use(NewAICachePlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.Set(aigwctx.TokenUsageKey, &token.TokenUsage{TotalTokens: 88})
		c.Data(http.StatusOK, "application/json", []byte(`{"origin":true}`))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"model":"m2","prompt":"hello"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-AI-Cache"); got != "MISS" {
		t.Fatalf("expected X-AI-Cache MISS, got %s", got)
	}
	if mock.setCalls != 1 {
		t.Fatalf("expected cache set once, got %d", mock.setCalls)
	}
	if mock.lastSet.key != "cache-key-1" {
		t.Fatalf("unexpected set key: %s", mock.lastSet.key)
	}
	if string(mock.lastSet.body) != `{"origin":true}` {
		t.Fatalf("unexpected set body: %s", string(mock.lastSet.body))
	}
	if mock.lastSet.token != 88 {
		t.Fatalf("expected token count 88, got %d", mock.lastSet.token)
	}
}

func TestAICachePluginGenerateKeyErrorPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAICachePluginTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableCache: true,
		},
	})

	mock := &mockStringCache{
		generateErr: errors.New("bad request"),
	}
	aiStringCacheGetter = func() stringCacheLike { return mock }

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(aigwctx.ConsumerNameKey, "consumer-c")
		c.Next()
	})
	r.Use(NewAICachePlugin().Handler())
	r.POST("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(`{"origin":true}`))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"prompt":"oops"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-AI-Cache"); got != "" {
		t.Fatalf("expected no cache header, got %s", got)
	}
	if mock.setCalls != 0 {
		t.Fatalf("expected no cache set on key generation failure")
	}
}

func prepareAICachePluginTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevGetter := aiStringCacheGetter
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiStringCacheGetter = prevGetter
	}
}
