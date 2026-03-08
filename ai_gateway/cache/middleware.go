package cache

import "github.com/gin-gonic/gin"

// AICacheMiddleware AI缓存中间件（待实现）
func AICacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
