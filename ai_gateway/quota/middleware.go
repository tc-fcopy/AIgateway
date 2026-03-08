package quota

import "github.com/gin-gonic/gin"

// QuotaMiddleware 配额中间件（待实现）
func QuotaMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
