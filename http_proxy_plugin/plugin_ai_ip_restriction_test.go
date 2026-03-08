package http_proxy_plugin

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"gateway/ai_gateway"
	aiconfig "gateway/ai_gateway/config"
	"gateway/ai_gateway/security"
	"github.com/gin-gonic/gin"
)

func TestAIIPRestrictionPluginAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIIPRestrictionTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableIPRestriction: true,
		},
		IPRestriction: aiconfig.IPRestrictionConfig{
			EnableCIDR: true,
			Whitelist:  []string{"192.0.2.10"},
			Blacklist:  []string{},
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "192.0.2.10:1234"

	plugin := NewAIIPRestrictionPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected request allowed by ip restriction")
	}
}

func TestAIIPRestrictionPluginDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIIPRestrictionTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableIPRestriction: true,
		},
		IPRestriction: aiconfig.IPRestrictionConfig{
			EnableCIDR: true,
			Whitelist:  []string{"192.0.2.10"},
			Blacklist:  []string{},
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "203.0.113.9:1234"

	plugin := NewAIIPRestrictionPlugin()
	result := plugin.Execute(NewExecContext(c))
	if !result.IsAbort() {
		t.Fatalf("expected ip restriction to deny request")
	}

	resp := map[string]interface{}{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode json body failed: %v", err)
	}
	if code, _ := resp["errno"].(float64); int(code) != 3601 {
		t.Fatalf("expected errno 3601, got %#v", resp["errno"])
	}
}

func prepareAIIPRestrictionTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevManager := ai_gateway.GlobalIPRestrictionManager
	ai_gateway.GlobalIPRestrictionManager = security.NewIPRestrictionManager()

	return func() {
		ai_gateway.GlobalIPRestrictionManager = prevManager
		aiconfig.AIConfManager.SetConfig(prevConf)
	}
}
