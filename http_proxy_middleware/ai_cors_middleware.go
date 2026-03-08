package http_proxy_middleware

import (
	"strconv"
	"strings"

	"gateway/ai_gateway/config"
	"github.com/gin-gonic/gin"
)

// AICORSMiddleware applies CORS policy for AI endpoints.
func AICORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsCORSEnabled() {
			c.Next()
			return
		}

		corsConf := config.AIConfManager.GetConfig().CORS
		origin := c.Request.Header.Get("Origin")

		if c.Request.Method == "OPTIONS" {
			if origin != "" && aiCORSOriginAllowed(origin, corsConf) {
				c.Header("Access-Control-Allow-Origin", origin)
			}
			if len(corsConf.AllowedMethods) > 0 {
				c.Header("Access-Control-Allow-Methods", strings.Join(corsConf.AllowedMethods, ", "))
			}
			if len(corsConf.AllowedHeaders) > 0 {
				c.Header("Access-Control-Allow-Headers", strings.Join(corsConf.AllowedHeaders, ", "))
			}
			if len(corsConf.ExposedHeaders) > 0 {
				c.Header("Access-Control-Expose-Headers", strings.Join(corsConf.ExposedHeaders, ", "))
			}
			if corsConf.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if corsConf.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", strconv.Itoa(corsConf.MaxAge))
			}

			c.AbortWithStatus(204)
			return
		}

		if origin != "" && aiCORSOriginAllowed(origin, corsConf) {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		if len(corsConf.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(corsConf.ExposedHeaders, ", "))
		}
		if corsConf.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Next()
	}
}

func aiCORSOriginAllowed(origin string, corsConf config.CORSConfig) bool {
	if corsConf.AllowAllOrigins {
		return true
	}

	if len(corsConf.AllowedOrigins) == 0 {
		return false
	}

	for _, allowed := range corsConf.AllowedOrigins {
		if allowed == "*" || origin == allowed {
			return true
		}

		if strings.HasSuffix(allowed, "*") {
			prefix := strings.TrimSuffix(allowed, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
	}

	return false
}
