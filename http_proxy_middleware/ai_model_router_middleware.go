package http_proxy_middleware

import (
	"encoding/json"

	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/model"
	"github.com/gin-gonic/gin"
)

// AIModelRouterMiddleware routes and maps model names in request payload.
func AIModelRouterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() {
			c.Next()
			return
		}

		if !config.AIConfManager.IsModelRouterEnabled() && !config.AIConfManager.IsModelMapperEnabled() {
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
			c.Next()
			return
		}

		originalModel := aiGetModel(payload, c)
		if originalModel == "" {
			c.Next()
			return
		}

		routedModel := originalModel
		if config.AIConfManager.IsModelRouterEnabled() {
			routedModel = model.GlobalModelRouter.Route(routedModel)
		}
		if config.AIConfManager.IsModelMapperEnabled() {
			routedModel = model.GlobalModelMapper.MapModel(routedModel)
		}

		c.Set(aigwctx.OriginalModelKey, originalModel)
		c.Set(aigwctx.ModelKey, routedModel)
		c.Set("ai_model", routedModel)
		c.Request.Header.Set("X-AI-Model", routedModel)

		if current, ok := payload["model"].(string); ok && current != routedModel {
			payload["model"] = routedModel
			updated, err := json.Marshal(payload)
			if err == nil {
				aiResetBody(c, updated)
			} else {
				aiResetBody(c, body)
			}
		} else {
			aiResetBody(c, body)
		}

		c.Next()
	}
}
