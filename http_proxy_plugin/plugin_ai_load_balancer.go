package http_proxy_plugin

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/loadbalancer"
	"github.com/gin-gonic/gin"
)

type aiLoadBalancerLike interface {
	SelectBackend() (*loadbalancer.Backend, error)
	ReleaseBackend(backendID string)
}

var aiLoadBalancerGetter = func() aiLoadBalancerLike {
	lb := ai_gateway.GetLoadBalancer()
	if lb == nil {
		lb = loadbalancer.LoadBalancer
	}
	return lb
}

// AILoadBalancerPlugin is the native plugin migration for ai.load_balancer.
// It keeps select-before-proxy and release-after-response semantics.
type AILoadBalancerPlugin struct{}

func NewAILoadBalancerPlugin() *AILoadBalancerPlugin {
	return &AILoadBalancerPlugin{}
}

func (p *AILoadBalancerPlugin) Name() string {
	return PluginAILoadBalancer
}

func (p *AILoadBalancerPlugin) Phase() Phase {
	return PhaseTraffic
}

func (p *AILoadBalancerPlugin) Priority() int {
	return 900
}

func (p *AILoadBalancerPlugin) Requires() []string {
	return nil
}

func (p *AILoadBalancerPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *AILoadBalancerPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}
	return Continue()
}

func (p *AILoadBalancerPlugin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.AIConfManager.IsEnabled() || !config.AIConfManager.IsLoadBalancerEnabled() {
			c.Next()
			return
		}

		lb := aiLoadBalancerGetter()
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
