package http_proxy_plugin

import (
	"encoding/json"
	"errors"

	"gateway/ai_gateway/config"
	"gateway/ai_gateway/prompt"
)

// AIPromptDecoratorPlugin is the native plugin migration for ai.prompt_decorator.
type AIPromptDecoratorPlugin struct{}

func NewAIPromptDecoratorPlugin() *AIPromptDecoratorPlugin {
	return &AIPromptDecoratorPlugin{}
}

func (p *AIPromptDecoratorPlugin) Name() string {
	return PluginAIPromptDecorator
}

func (p *AIPromptDecoratorPlugin) Phase() Phase {
	return PhaseTransform
}

func (p *AIPromptDecoratorPlugin) Priority() int {
	return 900
}

func (p *AIPromptDecoratorPlugin) Requires() []string {
	return nil
}

func (p *AIPromptDecoratorPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIPromptDecoratorPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsPromptDecoratorEnabled() {
		return Continue()
	}

	body, err := pluginReadBody(ctx.Gin)
	if err != nil {
		return Continue()
	}

	payload, err := pluginParseJSONBody(body)
	if err != nil {
		pluginResetBody(ctx.Gin, body)
		return Continue()
	}

	conf := config.AIConfManager.GetConfig()
	if conf != nil {
		prompt.GlobalPromptDecorator.SetConfig(
			conf.Enable && conf.DefaultService.EnablePromptDecorator,
			conf.PromptDecorator.SystemPrefix,
			conf.PromptDecorator.SystemSuffix,
			conf.PromptDecorator.UserPrefix,
			conf.PromptDecorator.UserSuffix,
		)
	}

	originalPrompt := pluginExtractPrompt(payload)
	if originalPrompt == "" {
		pluginResetBody(ctx.Gin, body)
		return Continue()
	}

	modelName := pluginGetModel(payload, ctx)
	decoratedPrompt, err := prompt.GlobalPromptDecorator.Decorate(originalPrompt, modelName)
	if err != nil {
		pluginResetBody(ctx.Gin, body)
		return Continue()
	}

	if _, ok := payload["prompt"]; ok {
		payload["prompt"] = decoratedPrompt
	}
	if messages, ok := payload["messages"].([]interface{}); ok {
		for i := len(messages) - 1; i >= 0; i-- {
			msg, ok := messages[i].(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role == "user" {
				msg["content"] = decoratedPrompt
				break
			}
		}
	}

	updated, err := json.Marshal(payload)
	if err != nil {
		pluginResetBody(ctx.Gin, body)
		return Continue()
	}

	ctx.Gin.Set("original_prompt", originalPrompt)
	ctx.Gin.Set("decorated_prompt", decoratedPrompt)
	pluginResetBody(ctx.Gin, updated)
	return Continue()
}

func pluginExtractPrompt(payload map[string]interface{}) string {
	if p, ok := payload["prompt"].(string); ok && p != "" {
		return p
	}

	if messages, ok := payload["messages"].([]interface{}); ok {
		for i := len(messages) - 1; i >= 0; i-- {
			msg, ok := messages[i].(map[string]interface{})
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			if role != "user" {
				continue
			}
			if content, ok := msg["content"].(string); ok {
				return content
			}
		}
	}

	if input, ok := payload["input"].(string); ok {
		return input
	}

	return ""
}
