package http_proxy_middleware

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

// AIIPRestrictionMiddleware enforces IP allow/deny rules.
func AIIPRestrictionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsIPRestrictionEnabled() {
			c.Next()
			return
		}

		conf := config.AIConfManager.GetConfig()
		if conf == nil {
			c.Next()
			return
		}

		manager := ai_gateway.GetIPRestrictionManager()
		if manager == nil {
			c.Next()
			return
		}

		manager.SetGlobalRules(conf.IPRestriction.EnableCIDR, conf.IPRestriction.Whitelist, conf.IPRestriction.Blacklist)

		ip := c.ClientIP()
		consumerName := c.GetString(aigwctx.ConsumerNameKey)
		if !manager.IsAllowed(ip, consumerName) {
			middleware.ResponseError(c, 3601, errors.New("ip is not allowed"))
			return
		}

		c.Next()
	}
}
