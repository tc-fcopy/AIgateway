package ratelimit

import "github.com/gin-gonic/gin"

// TokenRateLimitMiddleware Token限流中间件（待实现）
func TokenRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
