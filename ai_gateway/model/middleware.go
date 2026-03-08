package model

import "github.com/gin-gonic/gin"

// ModelRouterMiddleware 模型路由中间件（待实现）
func ModelRouterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
