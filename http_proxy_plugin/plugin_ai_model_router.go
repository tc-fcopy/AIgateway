package http_proxy_plugin

import (
	"encoding/json"
	"errors"

	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/model"
)

// AIModelRouterPlugin is the native plugin migration for ai.model_router.
type AIModelRouterPlugin struct{}

func NewAIModelRouterPlugin() *AIModelRouterPlugin {
	return &AIModelRouterPlugin{}
}

func (p *AIModelRouterPlugin) Name() string {
	return PluginAIModelRouter
}

func (p *AIModelRouterPlugin) Phase() Phase {
	return PhaseTransform
}

func (p *AIModelRouterPlugin) Priority() int {
	return 1000
}

func (p *AIModelRouterPlugin) Requires() []string {
	return nil
}

func (p *AIModelRouterPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIModelRouterPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	if !config.AIConfManager.IsEnabled() {
		return Continue()
	}

	if !config.AIConfManager.IsModelRouterEnabled() && !config.AIConfManager.IsModelMapperEnabled() {
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

	originalModel := pluginGetModel(payload, ctx)
	if originalModel == "" {
		pluginResetBody(ctx.Gin, body)
		return Continue()
	}

	routedModel := originalModel
	if config.AIConfManager.IsModelRouterEnabled() {
		routedModel = model.GlobalModelRouter.Route(routedModel)
	}
	if config.AIConfManager.IsModelMapperEnabled() {
		routedModel = model.GlobalModelMapper.MapModel(routedModel)
	}

	ctx.Gin.Set(aigwctx.OriginalModelKey, originalModel)
	ctx.Gin.Set(aigwctx.ModelKey, routedModel)
	ctx.Gin.Set("ai_model", routedModel)
	ctx.Gin.Request.Header.Set("X-AI-Model", routedModel)

	if current, ok := payload["model"].(string); ok && current != routedModel {
		payload["model"] = routedModel
		updated, mErr := json.Marshal(payload)
		if mErr == nil {
			pluginResetBody(ctx.Gin, updated)
			return Continue()
		}
	}

	pluginResetBody(ctx.Gin, body)
	return Continue()
}

func pluginParseJSONBody(body []byte) (map[string]interface{}, error) {
	if len(body) == 0 {
		return map[string]interface{}{}, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func pluginGetModel(payload map[string]interface{}, ctx *ExecContext) string {
	if payload != nil {
		if m, ok := payload["model"].(string); ok && m != "" {
			return m
		}
	}
	if m, ok := ctx.Gin.Get("ai_model"); ok {
		if modelName, ok := m.(string); ok && modelName != "" {
			return modelName
		}
	}
	return ctx.Gin.Query("model")
}
