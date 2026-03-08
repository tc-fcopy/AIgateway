package token

import "github.com/gin-gonic/gin"

// TokenMiddleware Token计量中间件（待实现）
func TokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
