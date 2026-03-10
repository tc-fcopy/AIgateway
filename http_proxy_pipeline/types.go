package http_proxy_pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// Phase defines logical execution stages for plugins.
type Phase int

const (
	PhasePreflight Phase = iota
	PhaseEdgeGuard
	PhaseAuthN
	PhasePolicy
	PhaseTransform
	PhaseTraffic
	PhaseObserve
	PhaseProxy
)

const (
	PluginAICORS            = "ai.cors"
	PluginCoreFlowCount     = "core.flow_count"
	PluginCoreFlowLimit     = "core.flow_limit"
	PluginCoreWhiteList     = "core.white_list"
	PluginCoreBlackList     = "core.black_list"
	PluginAIAuth            = "ai.auth"
	PluginAIIPRestriction   = "ai.ip_restriction"
	PluginAIModelRouter     = "ai.model_router"
	PluginAIPromptDecorator = "ai.prompt_decorator"
	PluginAITokenRateLimit  = "ai.token_ratelimit"
	PluginAIQuota           = "ai.quota"
	PluginAICache           = "ai.cache"
	PluginAILoadBalancer    = "ai.load_balancer"
	PluginAIObservability   = "ai.observability"
	PluginProxyHeader       = "proxy.header_transfer"
	PluginProxyStripURI     = "proxy.strip_uri"
	PluginProxyURLRewrite   = "proxy.url_rewrite"
	PluginProxyReverseProxy = "proxy.reverse_proxy"
)

// PlanContext is the per-service context used to build execution plans.
type PlanContext struct {
	ServiceID   int64
	ServiceName string
	LoadType    int

	AIEnabled         bool
	EnableCORS        bool
	EnableAuth        bool
	EnableIPRestrict  bool
	EnableModelRouter bool
	EnablePrompt      bool
	EnableTokenLimit  bool
	EnableQuota       bool
	EnableCache       bool
	EnableLoadBalance bool
	EnableObserve     bool

	StrictDependency  bool
	PriorityOverrides map[string]int
}

// ConfigVersion returns a compact fingerprint for plan cache keying.
func (p *PlanContext) ConfigVersion() string {
	if p == nil {
		return "nil"
	}
	toBit := func(v bool) string {
		if v {
			return "1"
		}
		return "0"
	}
	return strings.Join([]string{
		toBit(p.AIEnabled),
		toBit(p.EnableCORS),
		toBit(p.EnableAuth),
		toBit(p.EnableIPRestrict),
		toBit(p.EnableModelRouter),
		toBit(p.EnablePrompt),
		toBit(p.EnableTokenLimit),
		toBit(p.EnableQuota),
		toBit(p.EnableCache),
		toBit(p.EnableLoadBalance),
		toBit(p.EnableObserve),
		toBit(p.StrictDependency),
		mapSignature(p.PriorityOverrides),
	}, "")
}

func mapSignature(m map[string]int) string {
	if len(m) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ",")
}

// PluginSpec defines a plugin declaration for planning.
type PluginSpec struct {
	Name     string
	Phase    Phase
	Priority int
	Enabled  func(*PlanContext) bool
	Requires []string
}

// Plan is the compiled execution plan for a service.
type Plan struct {
	ServiceID         int64
	ServiceName       string
	ConfigVersion     string
	Plugins           []string
	Warnings          []string
	pluginSet         map[string]struct{}
	CompiledHandler   gin.HandlerFunc    // 预编译的执行链
	CompiledHandlers  []gin.HandlerFunc  // 调试用：单独的 handler 链
}

func (p *Plan) Has(pluginName string) bool {
	if p == nil {
		return false
	}
	_, ok := p.pluginSet[pluginName]
	return ok
}
