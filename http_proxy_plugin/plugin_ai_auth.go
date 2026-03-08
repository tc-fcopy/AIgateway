package http_proxy_plugin

import (
	"errors"
	"strings"

	"gateway/ai_gateway/auth"
	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
)

// AIAuthPlugin is the native plugin migration for ai.auth.
// Behavior keeps parity with legacy AIAuthMiddleware:
// JWT-first, API key fallback only when JWT token is absent.
type AIAuthPlugin struct{}

func NewAIAuthPlugin() *AIAuthPlugin {
	return &AIAuthPlugin{}
}

func (p *AIAuthPlugin) Name() string {
	return PluginAIAuth
}

func (p *AIAuthPlugin) Phase() Phase {
	return PhaseAuthN
}

func (p *AIAuthPlugin) Priority() int {
	return 1000
}

func (p *AIAuthPlugin) Requires() []string {
	return nil
}

func (p *AIAuthPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AIAuthPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	if !config.AIConfManager.IsEnabled() {
		return Continue()
	}

	conf := config.AIConfManager.GetConfig()
	if conf == nil {
		return Continue()
	}

	jwtEnabled := conf.DefaultService.EnableJWTAuth
	keyEnabled := conf.DefaultService.EnableKeyAuth
	if !jwtEnabled && !keyEnabled {
		return Continue()
	}

	if ctx.Gin.GetString(aigwctx.ConsumerNameKey) != "" {
		return Continue()
	}

	if jwtEnabled {
		if token, hasJWT := pluginExtractJWTToken(ctx, conf.JWTAuth); hasJWT {
			jwtAuth := auth.NewJWTAuth(conf.JWTAuth.Secret, conf.JWTAuth.Algorithms)
			claims, err := jwtAuth.Authenticate(token)
			if err != nil {
				middleware.ResponseError(ctx.Gin, 3011, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}

			consumerName := claims.Subject
			if consumerName == "" {
				err := errors.New("JWT token missing consumer identifier")
				middleware.ResponseError(ctx.Gin, 3012, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}

			cons, ok := consumer.ConsumerManager.GetByName(consumerName)
			if !ok {
				err := errors.New("consumer not found")
				middleware.ResponseError(ctx.Gin, 3013, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}
			if !cons.IsEnabled() {
				err := errors.New("consumer is disabled")
				middleware.ResponseError(ctx.Gin, 3014, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}
			if !cons.IsJWTType() {
				err := errors.New("invalid consumer type for JWT auth")
				middleware.ResponseError(ctx.Gin, 3015, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}
			if !pluginIsConsumerAllowed(cons.Name, conf.JWTAuth.AllowConsumers) {
				err := errors.New("consumer not allowed")
				middleware.ResponseError(ctx.Gin, 3016, err)
				return AbortWithStatus(ctx.Gin.Writer.Status(), err)
			}

			pluginBindConsumerContext(ctx, cons, claims)
			return Continue()
		}
	}

	if keyEnabled {
		apiKey, hasAPIKey := pluginExtractAPIKey(ctx, conf.KeyAuth.KeyNames)
		if !hasAPIKey {
			msg := "missing API key"
			if jwtEnabled {
				msg = "missing authentication credential"
			}
			err := errors.New(msg)
			middleware.ResponseError(ctx.Gin, 3002, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}

		cons, ok := consumer.ConsumerManager.GetByCredential(apiKey)
		if !ok {
			err := errors.New("invalid API key")
			middleware.ResponseError(ctx.Gin, 3003, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}
		if !cons.IsEnabled() {
			err := errors.New("consumer is disabled")
			middleware.ResponseError(ctx.Gin, 3004, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}
		if !cons.IsKeyType() {
			err := errors.New("invalid consumer type for key auth")
			middleware.ResponseError(ctx.Gin, 3005, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}
		if !pluginIsConsumerAllowed(cons.Name, conf.KeyAuth.AllowConsumers) {
			err := errors.New("consumer not allowed")
			middleware.ResponseError(ctx.Gin, 3006, err)
			return AbortWithStatus(ctx.Gin.Writer.Status(), err)
		}

		pluginBindConsumerContext(ctx, cons, nil)
		return Continue()
	}

	err := errors.New("missing JWT token")
	middleware.ResponseError(ctx.Gin, 3010, err)
	return AbortWithStatus(ctx.Gin.Writer.Status(), err)
}

func pluginBindConsumerContext(ctx *ExecContext, cons *consumer.Consumer, claims *auth.JWTClaims) {
	ctx.Gin.Set(aigwctx.ConsumerKey, cons)
	ctx.Gin.Set(aigwctx.ConsumerNameKey, cons.Name)
	if claims != nil {
		ctx.Gin.Set(aigwctx.JWTClaimsKey, claims)
	}

	ctx.Gin.Request.Header.Set("X-Consumer-Name", cons.Name)
	ctx.Gin.Request.Header.Set("X-Consumer-Type", cons.Type)
}

func pluginExtractJWTToken(ctx *ExecContext, conf config.JWTAuthConfig) (string, bool) {
	authorization := strings.TrimSpace(ctx.Gin.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return authorization, true
	}

	if conf.TokenHeader != "" {
		token := strings.TrimSpace(ctx.Gin.GetHeader(conf.TokenHeader))
		if token != "" {
			return token, true
		}
	}

	if conf.TokenQueryParam != "" {
		token := strings.TrimSpace(ctx.Gin.Query(conf.TokenQueryParam))
		if token != "" {
			return token, true
		}
	}

	return "", false
}

func pluginExtractAPIKey(ctx *ExecContext, keyNames []string) (string, bool) {
	for _, keyName := range keyNames {
		value := strings.TrimSpace(ctx.Gin.GetHeader(keyName))
		if value == "" {
			value = strings.TrimSpace(ctx.Gin.Query(keyName))
		}
		if value == "" {
			continue
		}

		if strings.EqualFold(keyName, "authorization") && strings.HasPrefix(strings.ToLower(value), "bearer ") {
			continue
		}
		return value, true
	}
	return "", false
}

func pluginIsConsumerAllowed(consumerName string, allowedConsumers []string) bool {
	if len(allowedConsumers) == 0 {
		return true
	}
	for _, allowed := range allowedConsumers {
		if allowed == "*" || allowed == consumerName {
			return true
		}
	}
	return false
}
