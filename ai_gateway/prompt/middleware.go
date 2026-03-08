package prompt

import "github.com/gin-gonic/gin"

// PromptDecoratorMiddleware placeholder for package-level middleware hook.
func PromptDecoratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
