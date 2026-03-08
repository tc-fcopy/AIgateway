package http_proxy_plugin

import (
	"errors"
	"net/http"

	"gateway/dao"
	"gateway/middleware"
	"gateway/reverse_proxy"
	"gateway/reverse_proxy/load_balance"
	"github.com/gin-gonic/gin"
)

type reverseProxyLike interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

var serviceLoadBalancerGetter = func(service *dao.ServiceDetail) (load_balance.LoadBalance, error) {
	return dao.LoadBalancerHandler.GetLoadBalancer(service)
}

var serviceTransportGetter = func(service *dao.ServiceDetail) (*http.Transport, error) {
	return dao.TransportorHandler.GetTrans(service)
}

var reverseProxyBuilder = func(c *gin.Context, lb load_balance.LoadBalance, trans *http.Transport) reverseProxyLike {
	return reverse_proxy.NewLoadBalanceReverseProxy(c, lb, trans)
}

// ProxyReverseProxyPlugin is the native plugin migration for proxy.reverse_proxy.
type ProxyReverseProxyPlugin struct{}

func NewProxyReverseProxyPlugin() *ProxyReverseProxyPlugin {
	return &ProxyReverseProxyPlugin{}
}

func (p *ProxyReverseProxyPlugin) Name() string {
	return PluginProxyReverseProxy
}

func (p *ProxyReverseProxyPlugin) Phase() Phase {
	return PhaseProxy
}

func (p *ProxyReverseProxyPlugin) Priority() int {
	return 1000
}

func (p *ProxyReverseProxyPlugin) Requires() []string {
	return nil
}

func (p *ProxyReverseProxyPlugin) Enabled(*ExecContext) bool {
	return true
}

func (p *ProxyReverseProxyPlugin) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}
	return Continue()
}

func (p *ProxyReverseProxyPlugin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		serviceDetail, ok := serverInterface.(*dao.ServiceDetail)
		if !ok || serviceDetail == nil {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}

		lb, err := serviceLoadBalancerGetter(serviceDetail)
		if err != nil {
			middleware.ResponseError(c, 2002, err)
			c.Abort()
			return
		}

		trans, err := serviceTransportGetter(serviceDetail)
		if err != nil {
			middleware.ResponseError(c, 2003, err)
			c.Abort()
			return
		}

		proxy := reverseProxyBuilder(c, lb, trans)
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
