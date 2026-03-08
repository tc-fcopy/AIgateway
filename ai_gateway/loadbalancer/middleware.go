package loadbalancer

import "github.com/gin-gonic/gin"

// AILoadBalancerMiddleware AI负载均衡中间件（待实现）
func AILoadBalancerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
