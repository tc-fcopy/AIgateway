package http_proxy_plugin

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"github.com/gin-gonic/gin"
)

func TestAIModelRouterPluginRouteAndMap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIModelRouterTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableModelRouter: true,
			EnableModelMapper: true,
		},
		ModelRouter: aiconfig.ModelRouterConfig{
			DefaultModel: "gpt-4o",
			Rules: []aiconfig.ModelRule{
				{Pattern: "gpt-4*", TargetModel: "gpt-4o"},
			},
		},
		ModelMapper: aiconfig.ModelMapperConfig{
			Mappings: []aiconfig.ModelMapping{
				{Source: "gpt-4o", Target: "vendor/gpt-4o"},
			},
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat", strings.NewReader(`{"model":"gpt-4-turbo","prompt":"hi"}`))

	plugin := NewAIModelRouterPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	if got := c.GetString(aigwctx.OriginalModelKey); got != "gpt-4-turbo" {
		t.Fatalf("unexpected original model: %s", got)
	}
	if got := c.GetString(aigwctx.ModelKey); got != "vendor/gpt-4o" {
		t.Fatalf("unexpected routed model: %s", got)
	}
	if got := c.Request.Header.Get("X-AI-Model"); got != "vendor/gpt-4o" {
		t.Fatalf("unexpected header model: %s", got)
	}

	body, err := pluginReadBody(c)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	payload := map[string]interface{}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if payload["model"] != "vendor/gpt-4o" {
		t.Fatalf("expected body model rewritten, got %#v", payload["model"])
	}
}

func TestAIModelRouterPluginInvalidJSONPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIModelRouterTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableModelRouter: true,
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat", strings.NewReader("{invalid"))

	plugin := NewAIModelRouterPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	body, err := pluginReadBody(c)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if string(body) != "{invalid" {
		t.Fatalf("expected body unchanged, got %s", string(body))
	}
}

func prepareAIModelRouterTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
	}
}
