package security

import "github.com/gin-gonic/gin"

// IPRestrictionMiddleware placeholder for package-level middleware hook.
func IPRestrictionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
