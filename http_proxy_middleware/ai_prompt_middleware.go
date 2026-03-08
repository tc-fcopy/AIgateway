package http_proxy_middleware

import (
	"encoding/json"

	"gateway/ai_gateway/config"
	"gateway/ai_gateway/prompt"
	"github.com/gin-gonic/gin"
)

// AIPromptMiddleware decorates prompts based on config.
func AIPromptMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsPromptDecoratorEnabled() {
			c.Next()
			return
		}

		body, err := aiReadBody(c)
		if err != nil {
			c.Next()
			return
		}

		payload, err := aiParseJSONBody(body)
		if err != nil {
			aiResetBody(c, body)
			c.Next()
			return
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

		originalPrompt := aiExtractPrompt(payload)
		if originalPrompt == "" {
			aiResetBody(c, body)
			c.Next()
			return
		}

		modelName := aiGetModel(payload, c)
		decoratedPrompt, err := prompt.GlobalPromptDecorator.Decorate(originalPrompt, modelName)
		if err != nil {
			aiResetBody(c, body)
			c.Next()
			return
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
			aiResetBody(c, body)
			c.Next()
			return
		}

		c.Set("original_prompt", originalPrompt)
		c.Set("decorated_prompt", decoratedPrompt)
		aiResetBody(c, updated)
		c.Next()
	}
}
