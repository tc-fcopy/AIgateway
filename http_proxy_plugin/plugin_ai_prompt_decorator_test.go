package http_proxy_plugin

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	"gateway/ai_gateway/prompt"
	"github.com/gin-gonic/gin"
)

func TestAIPromptDecoratorPluginDecoratePrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIPromptDecoratorTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnablePromptDecorator: true,
		},
		PromptDecorator: aiconfig.PromptDecoratorConfig{
			SystemPrefix: "[SYS]",
			SystemSuffix: "[/SYS]",
			UserPrefix:   "<U>",
			UserSuffix:   "</U>",
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat", strings.NewReader(`{"model":"gpt-4o","prompt":"hello"}`))

	plugin := NewAIPromptDecoratorPlugin()
	result := plugin.Execute(NewExecContext(c))
	if result.IsAbort() {
		t.Fatalf("expected continue result")
	}

	expectedPrompt := "[SYS]\n<U>hello</U>\n[/SYS]"
	if got := c.GetString("original_prompt"); got != "hello" {
		t.Fatalf("unexpected original prompt: %s", got)
	}
	if got := c.GetString("decorated_prompt"); got != expectedPrompt {
		t.Fatalf("unexpected decorated prompt: %s", got)
	}

	body, err := pluginReadBody(c)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}

	payload := map[string]interface{}{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
	if payload["prompt"] != expectedPrompt {
		t.Fatalf("expected prompt rewritten, got %#v", payload["prompt"])
	}
}

func TestAIPromptDecoratorPluginInvalidJSONPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIPromptDecoratorTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnablePromptDecorator: true,
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat", strings.NewReader("{invalid"))

	plugin := NewAIPromptDecoratorPlugin()
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

func prepareAIPromptDecoratorTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		if prevConf != nil {
			prompt.GlobalPromptDecorator.SetConfig(
				prevConf.Enable && prevConf.DefaultService.EnablePromptDecorator,
				prevConf.PromptDecorator.SystemPrefix,
				prevConf.PromptDecorator.SystemSuffix,
				prevConf.PromptDecorator.UserPrefix,
				prevConf.PromptDecorator.UserSuffix,
			)
			return
		}
		prompt.GlobalPromptDecorator.SetConfig(true, "", "", "", "")
	}
}
