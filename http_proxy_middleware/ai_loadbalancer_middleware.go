package http_proxy_middleware

import (
	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/loadbalancer"
	"github.com/gin-gonic/gin"
)

// AILoadBalancerMiddleware selects backend node for AI request.
func AILoadBalancerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsLoadBalancerEnabled() {
			c.Next()
			return
		}

		lb := ai_gateway.GetLoadBalancer()
		if lb == nil {
			lb = loadbalancer.LoadBalancer
		}
		if lb == nil {
			c.Next()
			return
		}

		backend, err := lb.SelectBackend()
		if err != nil || backend == nil {
			c.Next()
			return
		}

		c.Set(aigwctx.BackendIDKey, backend.ID)
		c.Set(aigwctx.BackendAddressKey, backend.Address)
		c.Request.Header.Set("X-Backend-ID", backend.ID)
		c.Request.Header.Set("X-Backend-Address", backend.Address)

		defer lb.ReleaseBackend(backend.ID)
		c.Next()
	}
}

func GetBackendFromContext(c *gin.Context) (string, bool) {
	backendID, exists := c.Get(aigwctx.BackendIDKey)
	if !exists {
		return "", false
	}
	id, ok := backendID.(string)
	return id, ok
}

func GetBackendAddressFromContext(c *gin.Context) (string, bool) {
	address, exists := c.Get(aigwctx.BackendAddressKey)
	if !exists {
		return "", false
	}
	addr, ok := address.(string)
	return addr, ok
}
