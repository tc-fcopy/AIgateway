package http_proxy_middleware

import (
	"errors"
	"strings"

	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

// AIKeyAuthMiddleware authenticates by api key and sets consumer context.
func AIKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsKeyAuthEnabled() {
			c.Next()
			return
		}

		conf := config.AIConfManager.GetConfig()
		if conf == nil {
			c.Next()
			return
		}

		apiKey := ""
		for _, keyName := range conf.KeyAuth.KeyNames {
			apiKey = c.GetHeader(keyName)
			if apiKey == "" {
				apiKey = c.Query(keyName)
			}
			if apiKey != "" {
				if strings.EqualFold(keyName, "authorization") {
					apiKey = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(apiKey, "Bearer "), "bearer "))
				}
				break
			}
		}

		if apiKey == "" {
			middleware.ResponseError(c, 3002, errors.New("missing API key"))
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

		c.Set(aigwctx.ConsumerKey, cons)
		c.Set(aigwctx.ConsumerNameKey, cons.Name)
		c.Request.Header.Set("X-Consumer-Name", cons.Name)
		c.Request.Header.Set("X-Consumer-Type", cons.Type)

		c.Next()
	}
}
