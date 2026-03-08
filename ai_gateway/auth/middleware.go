package auth

import (
	"errors"

	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

func KeyAuthMiddleware() gin.HandlerFunc {
	keyAuth := NewKeyAuth()
	return func(c *gin.Context) {
		cons, err := keyAuth.AuthenticateWithContext(c)
		if err != nil {
			middleware.ResponseError(c, 3000, err)
			c.Abort()
			return
		}

		c.Set(aigwctx.ConsumerKey, cons)
		c.Set(aigwctx.ConsumerNameKey, cons.Name)

		c.Request.Header.Set("X-Consumer-Name", cons.Name)
		c.Request.Header.Set("X-Consumer-Type", cons.Type)
		c.Next()
	}
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() {
			c.Next()
			return
		}

		aiConfig := config.AIConfManager.GetConfig()
		if aiConfig == nil || !aiConfig.Enable || !aiConfig.DefaultService.EnableJWTAuth {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.GetHeader("authorization")
		}

		if authHeader == "" {
			middleware.ResponseError(c, 3102, errors.New("missing Authorization header"))
			c.Abort()
			return
		}

		jwtAuth := NewJWTAuth(aiConfig.JWTAuth.Secret, aiConfig.JWTAuth.Algorithms)
		claims, err := jwtAuth.Authenticate(authHeader)
		if err != nil {
			middleware.ResponseError(c, 3103, err)
			c.Abort()
			return
		}

		consumerName := jwtAuth.GetConsumerName(claims)
		cons, ok := consumer.ConsumerManager.GetByName(consumerName)
		if !ok {
			middleware.ResponseError(c, 3104, errors.New("consumer not found"))
			c.Abort()
			return
		}

		if !cons.IsEnabled() {
			middleware.ResponseError(c, 3105, errors.New("consumer is disabled"))
			c.Abort()
			return
		}

		if !cons.IsJWTType() {
			middleware.ResponseError(c, 3106, errors.New("invalid consumer type for JWT auth"))
			c.Abort()
			return
		}

		c.Set(aigwctx.ConsumerKey, cons)
		c.Set(aigwctx.ConsumerNameKey, cons.Name)
		c.Request.Header.Set("X-Consumer-Name", cons.Name)
		c.Request.Header.Set("X-Consumer-Type", cons.Type)

		c.Next()
	}
}

func GetConsumerFromContext(c *gin.Context) (*consumer.Consumer, bool) {
	cons, exists := c.Get(aigwctx.ConsumerKey)
	if !exists {
		return nil, false
	}
	out, ok := cons.(*consumer.Consumer)
	return out, ok
}

func GetConsumerNameFromContext(c *gin.Context) (string, bool) {
	name, exists := c.Get(aigwctx.ConsumerNameKey)
	if !exists {
		return "", false
	}
	out, ok := name.(string)
	return out, ok
}
