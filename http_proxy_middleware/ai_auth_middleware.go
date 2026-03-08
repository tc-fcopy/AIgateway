package http_proxy_middleware

import (
	"errors"
	"strings"

	"gateway/ai_gateway/auth"
	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

// AIAuthMiddleware authenticates AI requests with JWT-first priority, then API key fallback.
// This mirrors Higress-style behavior where JWT auth has higher priority than key auth.
func AIAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() {
			c.Next()
			return
		}

		conf := config.AIConfManager.GetConfig()
		if conf == nil {
			c.Next()
			return
		}

		jwtEnabled := conf.DefaultService.EnableJWTAuth
		keyEnabled := conf.DefaultService.EnableKeyAuth
		if !jwtEnabled && !keyEnabled {
			c.Next()
			return
		}

		// If an upstream middleware has already authenticated the request, skip.
		if c.GetString(aigwctx.ConsumerNameKey) != "" {
			c.Next()
			return
		}

		// 1) JWT has higher priority when a JWT token is provided.
		if jwtEnabled {
			if token, hasJWT := aiExtractJWTToken(c, conf.JWTAuth); hasJWT {
				jwtAuth := auth.NewJWTAuth(conf.JWTAuth.Secret, conf.JWTAuth.Algorithms)
				claims, err := jwtAuth.Authenticate(token)
				if err != nil {
					middleware.ResponseError(c, 3011, err)
					return
				}

				consumerName := claims.Subject
				if consumerName == "" {
					middleware.ResponseError(c, 3012, errors.New("JWT token missing consumer identifier"))
					return
				}

				cons, ok := consumer.ConsumerManager.GetByName(consumerName)
				if !ok {
					middleware.ResponseError(c, 3013, errors.New("consumer not found"))
					return
				}
				if !cons.IsEnabled() {
					middleware.ResponseError(c, 3014, errors.New("consumer is disabled"))
					return
				}
				if !cons.IsJWTType() {
					middleware.ResponseError(c, 3015, errors.New("invalid consumer type for JWT auth"))
					return
				}
				if !aiIsConsumerAllowed(cons.Name, conf.JWTAuth.AllowConsumers) {
					middleware.ResponseError(c, 3016, errors.New("consumer not allowed"))
					return
				}

				aiBindConsumerContext(c, cons, claims)
				c.Next()
				return
			}
		}

		// 2) API key fallback when no JWT token is provided.
		if keyEnabled {
			apiKey, hasAPIKey := aiExtractAPIKey(c, conf.KeyAuth.KeyNames)
			if !hasAPIKey {
				msg := "missing API key"
				if jwtEnabled {
					msg = "missing authentication credential"
				}
				middleware.ResponseError(c, 3002, errors.New(msg))
				return
			}

			cons, ok := consumer.ConsumerManager.GetByCredential(apiKey)
			if !ok {
				middleware.ResponseError(c, 3003, errors.New("invalid API key"))
				return
			}
			if !cons.IsEnabled() {
				middleware.ResponseError(c, 3004, errors.New("consumer is disabled"))
				return
			}
			if !cons.IsKeyType() {
				middleware.ResponseError(c, 3005, errors.New("invalid consumer type for key auth"))
				return
			}
			if !aiIsConsumerAllowed(cons.Name, conf.KeyAuth.AllowConsumers) {
				middleware.ResponseError(c, 3006, errors.New("consumer not allowed"))
				return
			}

			aiBindConsumerContext(c, cons, nil)
			c.Next()
			return
		}

		// JWT is enabled but no JWT token provided and key auth is disabled.
		middleware.ResponseError(c, 3010, errors.New("missing JWT token"))
	}
}

func aiBindConsumerContext(c *gin.Context, cons *consumer.Consumer, claims *auth.JWTClaims) {
	c.Set(aigwctx.ConsumerKey, cons)
	c.Set(aigwctx.ConsumerNameKey, cons.Name)
	if claims != nil {
		c.Set(aigwctx.JWTClaimsKey, claims)
	}

	c.Request.Header.Set("X-Consumer-Name", cons.Name)
	c.Request.Header.Set("X-Consumer-Type", cons.Type)
}

func aiExtractJWTToken(c *gin.Context, conf config.JWTAuthConfig) (string, bool) {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return authorization, true
	}

	if conf.TokenHeader != "" {
		token := strings.TrimSpace(c.GetHeader(conf.TokenHeader))
		if token != "" {
			return token, true
		}
	}

	if conf.TokenQueryParam != "" {
		token := strings.TrimSpace(c.Query(conf.TokenQueryParam))
		if token != "" {
			return token, true
		}
	}

	return "", false
}

func aiExtractAPIKey(c *gin.Context, keyNames []string) (string, bool) {
	for _, keyName := range keyNames {
		value := strings.TrimSpace(c.GetHeader(keyName))
		if value == "" {
			value = strings.TrimSpace(c.Query(keyName))
		}
		if value == "" {
			continue
		}

		// When Authorization carries Bearer token, it should be handled by JWT path.
		if strings.EqualFold(keyName, "authorization") && strings.HasPrefix(strings.ToLower(value), "bearer ") {
			continue
		}
		return value, true
	}
	return "", false
}
