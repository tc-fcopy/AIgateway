package http_proxy_plugin

import (
	"fmt"
	"sync"

	"gateway/http_proxy_middleware"
	"github.com/gin-gonic/gin"
)

var (
	builtinOnce sync.Once
	builtinErr  error
)

// RegisterBuiltinPlugins registers all built-in plugins into GlobalRegistry once.
func RegisterBuiltinPlugins() error {
	builtinOnce.Do(func() {
		builtinErr = registerBuiltinPlugins(GlobalRegistry)
	})
	return builtinErr
}

// RegisterBuiltinPluginsTo registers all built-in plugins into a custom registry.
func RegisterBuiltinPluginsTo(reg *Registry) error {
	if reg == nil {
		return ErrRegistryNil
	}
	return registerBuiltinPlugins(reg)
}

func registerBuiltinPlugins(reg *Registry) error {
	if err := reg.Register(NewCoreFlowCountPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginCoreFlowCount, err)
	}
	if err := reg.Register(NewCoreFlowLimitPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginCoreFlowLimit, err)
	}
	if err := reg.Register(NewCoreWhiteListPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginCoreWhiteList, err)
	}
	if err := reg.Register(NewCoreBlackListPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginCoreBlackList, err)
	}
	if err := reg.Register(NewAIAuthPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIAuth, err)
	}
	if err := reg.Register(NewAIIPRestrictionPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIIPRestriction, err)
	}
	if err := reg.Register(NewAITokenRateLimitPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAITokenRateLimit, err)
	}
	if err := reg.Register(NewAIQuotaPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIQuota, err)
	}
	if err := reg.Register(NewAIModelRouterPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIModelRouter, err)
	}
	if err := reg.Register(NewAIPromptDecoratorPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIPromptDecorator, err)
	}
	if err := reg.Register(NewAICachePlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAICache, err)
	}
	if err := reg.Register(NewAILoadBalancerPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAILoadBalancer, err)
	}
	if err := reg.Register(NewAIObservabilityPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginAIObservability, err)
	}
	if err := reg.Register(NewProxyHeaderTransferPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginProxyHeader, err)
	}
	if err := reg.Register(NewProxyStripURIPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginProxyStripURI, err)
	}
	if err := reg.Register(NewProxyURLRewritePlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginProxyURLRewrite, err)
	}
	if err := reg.Register(NewProxyReverseProxyPlugin()); err != nil {
		return fmt.Errorf("register plugin %s failed: %w", PluginProxyReverseProxy, err)
	}

	type item struct {
		spec    AdapterSpec
		factory func() gin.HandlerFunc
	}

	items := []item{
		{spec: AdapterSpec{Name: PluginAICORS, Phase: PhasePreflight, Priority: 1000}, factory: http_proxy_middleware.AICORSMiddleware},
	}

	for _, it := range items {
		plugin, err := NewMiddlewareAdapter(it.spec, it.factory())
		if err != nil {
			return fmt.Errorf("build plugin adapter %s failed: %w", it.spec.Name, err)
		}
		if err := reg.Register(plugin); err != nil {
			return fmt.Errorf("register plugin %s failed: %w", it.spec.Name, err)
		}
	}
	return nil
}
