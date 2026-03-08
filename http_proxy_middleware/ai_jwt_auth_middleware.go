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

// AIJWTAuthMiddleware authenticates by JWT and sets consumer context.
func AIJWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsJWTAuthEnabled() {
			c.Next()
			return
		}

		conf := config.AIConfManager.GetConfig()
		if conf == nil {
			c.Next()
			return
		}

		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if token == "" && conf.JWTAuth.TokenHeader != "" {
			token = strings.TrimSpace(c.GetHeader(conf.JWTAuth.TokenHeader))
		}
		if token == "" && conf.JWTAuth.TokenQueryParam != "" {
			token = strings.TrimSpace(c.Query(conf.JWTAuth.TokenQueryParam))
		}
		if token == "" {
			middleware.ResponseError(c, 3010, errors.New("missing JWT token"))
			return
		}

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

		c.Set(aigwctx.ConsumerKey, cons)
		c.Set(aigwctx.ConsumerNameKey, cons.Name)
		c.Set(aigwctx.JWTClaimsKey, claims)

		c.Request.Header.Set("X-Consumer-Name", cons.Name)
		c.Request.Header.Set("X-Consumer-Type", cons.Type)

		c.Next()
	}
}
