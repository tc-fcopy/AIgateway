package http_proxy_plugin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	aiconfig "gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	aigwctx "gateway/ai_gateway/context"
	"github.com/gin-gonic/gin"
)

func TestAIAuthPluginJWTFirstSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIAuthTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: true,
			EnableJWTAuth: true,
		},
		KeyAuth: aiconfig.KeyAuthConfig{
			KeyNames: []string{"X-API-Key"},
		},
		JWTAuth: aiconfig.JWTAuthConfig{
			Secret:     "test-secret",
			Algorithms: []string{"HS256"},
		},
	})
	consumer.ConsumerManager.LoadConsumers([]*consumer.Consumer{
		{Name: "jwt-user", Credential: "jwt-cred", Type: "jwt", Status: 1},
		{Name: "key-user", Credential: "key-cred", Type: "key", Status: 1},
	})

	token := buildHS256Token(t, "test-secret", "jwt-user", time.Now().Add(5*time.Minute).Unix())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	c.Request.Header.Set("X-API-Key", "key-cred")

	plugin := NewAIAuthPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected auth success, got abort")
	}

	if got := c.GetString(aigwctx.ConsumerNameKey); got != "jwt-user" {
		t.Fatalf("expected jwt consumer selected, got %s", got)
	}
	if got := c.Request.Header.Get("X-Consumer-Type"); got != "jwt" {
		t.Fatalf("expected consumer type jwt, got %s", got)
	}
}

func TestAIAuthPluginKeyFallbackSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIAuthTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: true,
			EnableJWTAuth: true,
		},
		KeyAuth: aiconfig.KeyAuthConfig{
			KeyNames: []string{"X-API-Key"},
		},
		JWTAuth: aiconfig.JWTAuthConfig{
			Secret:     "test-secret",
			Algorithms: []string{"HS256"},
		},
	})
	consumer.ConsumerManager.LoadConsumers([]*consumer.Consumer{
		{Name: "key-user", Credential: "key-cred", Type: "key", Status: 1},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "key-cred")

	plugin := NewAIAuthPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected key fallback success, got abort")
	}
	if got := c.GetString(aigwctx.ConsumerNameKey); got != "key-user" {
		t.Fatalf("expected key consumer selected, got %s", got)
	}
}

func TestAIAuthPluginJWTErrorNoFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIAuthTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth: true,
			EnableJWTAuth: true,
		},
		KeyAuth: aiconfig.KeyAuthConfig{
			KeyNames: []string{"X-API-Key"},
		},
		JWTAuth: aiconfig.JWTAuthConfig{
			Secret:     "test-secret",
			Algorithms: []string{"HS256"},
		},
	})
	consumer.ConsumerManager.LoadConsumers([]*consumer.Consumer{
		{Name: "key-user", Credential: "key-cred", Type: "key", Status: 1},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer bad.jwt.token")
	c.Request.Header.Set("X-API-Key", "key-cred")

	plugin := NewAIAuthPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected abort when jwt token invalid")
	}

	resp := decodeJSONBody(t, w.Body.Bytes())
	if code, _ := resp["errno"].(float64); int(code) != 3011 {
		t.Fatalf("expected errno 3011, got %#v", resp["errno"])
	}
	if got := c.GetString(aigwctx.ConsumerNameKey); got != "" {
		t.Fatalf("expected no consumer context on jwt error, got %s", got)
	}
}

func prepareAIAuthTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevMgr := consumer.ConsumerManager
	consumer.ConsumerManager = consumer.NewManager()

	return func() {
		consumer.ConsumerManager = prevMgr
		aiconfig.AIConfManager.SetConfig(prevConf)
	}
}

func buildHS256Token(t *testing.T, secret, subject string, exp int64) string {
	t.Helper()
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"sub": subject,
		"exp": exp,
		"iat": time.Now().Unix(),
	}

	enc := base64.RawURLEncoding
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	headerPart := enc.EncodeToString(hb)
	payloadPart := enc.EncodeToString(pb)
	signingInput := headerPart + "." + payloadPart

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	sig := enc.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s.%s.%s", headerPart, payloadPart, sig)
}

func decodeJSONBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	out := map[string]interface{}{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode json body failed: %v", err)
	}
	return out
}
