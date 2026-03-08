package http_proxy_middleware

import (
	"github.com/gin-gonic/gin"
)

// PlanAware is kept for backward compatibility.
//
// Since executor mode has taken over the proxy main chain, this wrapper now
// behaves as a no-op pass-through.
func PlanAware(pluginName string, handler gin.HandlerFunc) gin.HandlerFunc {
	_ = pluginName
	return handler
}
