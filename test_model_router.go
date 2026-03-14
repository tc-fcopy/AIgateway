package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"gateway/ai_gateway/config"
	"gateway/ai_gateway/model"
	"gateway/dao"
	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
)

func TestModelRouterPlugin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fmt.Println("=== 测试 ModelRouter 插件 ===")

	// 1. 配置 AI 配置
	prev := config.AIConfManager.GetConfig()
	defer config.AIConfManager.SetConfig(prev)

	config.AIConfManager.SetConfig(&config.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: config.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableModelRouter:     true,
			EnableModelMapper:     true,
		},
	})

	// 2. 配置模型路由规则
	model.GlobalModelRouter.SetConfig(true, "gpt-4", []model.ModelRule{
		{
			Pattern:     "gpt-*",
			TargetModel: "gpt-4",
			Priority:    100,
		},
		{
			Pattern:     "claude-*",
			TargetModel: "claude-3",
			Priority:    100,
		},
	})

	// 3. 配置模型映射
	model.GlobalModelMapper.SetConfig([]model.ModelMapping{
		{
			Source: "gpt-4",
			Target: "gpt-4-turbo",
		},
		{
			Source: "claude-3",
			Target: "claude-3-opus",
		},
	}, true)

	// 4. 创建测试上下文
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	testService := &dao.ServiceDetail{
		Info: &dao.ServiceInfo{
			ID:          1001,
			ServiceName: "test-ai-service",
		},
	}
	c.Set("service", testService)

	// 5. 注册插件
	http_proxy_plugin.RegisterBuiltinPluginsTo(http_proxy_plugin.GlobalRegistry)

	// 6. 创建测试请求体 - gpt-3.5-turbo
	fmt.Println("\n--- 测试 1: 发送 gpt-3.5-turbo ---")
	testGptRequest(t, c, "gpt-3.5-turbo", "Hello!")

	// 7. 重置上下文
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Set("service", testService)

	// 8. 创建测试请求体 - claude-2
	fmt.Println("\n--- 测试 2: 发送 claude-2 ---")
	testClaudeRequest(t, c, "claude-2", "Hello!")

	fmt.Println("\n=== 测试完成 ===")
}

func testGptRequest(t *testing.T, c *gin.Context, modelName string, content string) {
	// 创建请求体
	body := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	jsonBody, _ := json.Marshal(body)

	c.Request.Body = &mockReadCloser{bytes.NewReader(jsonBody)}
	originalBody := make([]byte, len(jsonBody))
	copy(originalBody, jsonBody)

	// 获取插件
	plugin, ok := http_proxy_plugin.GlobalRegistry.Get(http_proxy_plugin.PluginAIModelRouter)
	if !ok {
		t.Fatalf("Plugin not found: %s", http_proxy_plugin.PluginAIModelRouter)
	}

	// 执行插件
	ctx := http_proxy_plugin.NewExecContext(c)
	result := plugin.Execute(ctx)

	if result.IsAbort() {
		t.Fatalf("Plugin aborted: %v", result.Err)
	}

	// 检查结果
	// 1. 检查 X-AI-Model Header
	aiModel := c.GetHeader("X-AI-Model")
	fmt.Printf("X-AI-Model Header: %s\n", aiModel)
	if aiModel == "" {
		fmt.Println("❌ X-AI-Model Header not set")
	} else if aiModel != "gpt-4-turbo" {
		fmt.Printf("⚠️  X-AI-Model Header = %s (expected: gpt-4-turbo)\n", aiModel)
	} else {
		fmt.Println("✅ X-AI-Model Header = gpt-4-turbo")
	}

	// 2. 检查 context 中的 ai_model
	if ctxModel, ok := c.Get("ai_model"); ok {
		fmt.Printf("Context ai_model: %v\n", ctxModel)
	}

	fmt.Println("✅ gpt-3.5-turbo 测试完成")
}

func testClaudeRequest(t *testing.T, c *gin.Context, modelName string, content string) {
	// 创建请求体
	body := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	jsonBody, _ := json.Marshal(body)

	c.Request.Body = &mockReadCloser{bytes.NewReader(jsonBody)}
	originalBody := make([]byte, len(jsonBody))
	copy(originalBody, jsonBody)

	// 获取插件
	plugin, ok := http_proxy_plugin.GlobalRegistry.Get(http_proxy_plugin.PluginAIModelRouter)
	if !ok {
		t.Fatalf("Plugin not found: %s", http_proxy_plugin.PluginAIModelRouter)
	}

	// 执行插件
	ctx := http_proxy_plugin.NewExecContext(c)
	result := plugin.Execute(ctx)

	if result.IsAbort() {
		t.Fatalf("Plugin aborted: %v", result.Err)
	}

	// 检查结果
	aiModel := c.GetHeader("X-AI-Model")
	fmt.Printf("X-AI-Model Header: %s\n", aiModel)
	if aiModel == "" {
		fmt.Println("❌ X-AI-Model Header not set")
	} else if aiModel != "claude-3-opus" {
		fmt.Printf("⚠️  X-AI-Model Header = %s (expected: claude-3-opus)\n", aiModel)
	} else {
		fmt.Println("✅ X-AI-Model Header = claude-3-opus")
	}

	fmt.Println("✅ claude-2 测试完成")
}

func boolPtr(v bool) *bool {
	p := v
	return &p
}

type mockReadCloser struct {
	*bytes.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func main() {
	fmt.Println("运行 ModelRouter 测试...")
	TestModelRouterPlugin(&testing.T{})
}
