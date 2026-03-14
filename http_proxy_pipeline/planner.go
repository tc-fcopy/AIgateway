package http_proxy_pipeline

import (
	"fmt"
	"sort"

	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
)

func defaultPluginSpecs() []PluginSpec {
	isAI := func(pc *PlanContext) bool {
		return pc != nil && pc.AIEnabled
	}

	return []PluginSpec{
		{Name: PluginAICORS, Phase: PhasePreflight, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableCORS }},

		{Name: PluginCoreFlowCount, Phase: PhaseEdgeGuard, Priority: 900, Enabled: alwaysOn},
		{Name: PluginCoreFlowLimit, Phase: PhaseEdgeGuard, Priority: 800, Enabled: alwaysOn},
		{Name: PluginCoreIPACL, Phase: PhaseEdgeGuard, Priority: 700, Enabled: alwaysOn},

		{Name: PluginAIAuth, Phase: PhaseAuthN, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableAuth }},

		{Name: PluginAIIPRestriction, Phase: PhasePolicy, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableIPRestrict }},
		{Name: PluginAITokenRateLimit, Phase: PhasePolicy, Priority: 900, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableTokenLimit }, Requires: []string{PluginAIAuth}},
		{Name: PluginAIQuota, Phase: PhasePolicy, Priority: 800, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableQuota }, Requires: []string{PluginAIAuth}},

		{Name: PluginAIModelRouter, Phase: PhaseTransform, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableModelRouter }},
		{Name: PluginAIPromptDecorator, Phase: PhaseTransform, Priority: 900, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnablePrompt }},
		{Name: PluginProxyHeader, Phase: PhaseTransform, Priority: 500, Enabled: alwaysOn},
		{Name: PluginProxyStripURI, Phase: PhaseTransform, Priority: 400, Enabled: alwaysOn},
		{Name: PluginProxyURLRewrite, Phase: PhaseTransform, Priority: 300, Enabled: alwaysOn},

		{Name: PluginAICache, Phase: PhaseTraffic, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableCache }, Requires: []string{PluginAIAuth}},
		{Name: PluginAILoadBalancer, Phase: PhaseTraffic, Priority: 900, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableLoadBalance }},

		{Name: PluginAIObservability, Phase: PhaseObserve, Priority: 1000, Enabled: func(pc *PlanContext) bool { return isAI(pc) && pc.EnableObserve }},

		{Name: PluginProxyReverseProxy, Phase: PhaseProxy, Priority: 1000, Enabled: alwaysOn},
	}
}

func alwaysOn(_ *PlanContext) bool { return true }

func buildPlan(serviceID int64, serviceName string, pc *PlanContext, specs []PluginSpec, registry *http_proxy_plugin.Registry) *Plan {
	effective := applyPriorityOverrides(specs, pc.PriorityOverrides)

	enabled := make([]PluginSpec, 0, len(effective))
	for _, sp := range effective {
		if sp.Enabled == nil || sp.Enabled(pc) {
			enabled = append(enabled, sp)
		}
	}

	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].Phase != enabled[j].Phase {
			return enabled[i].Phase < enabled[j].Phase
		}
		return enabled[i].Priority > enabled[j].Priority
	})

	validated, warnings := validateDependencies(enabled, pc.StrictDependency)

	plugins := make([]string, 0, len(validated))
	set := make(map[string]struct{}, len(validated))
	for _, sp := range validated {
		plugins = append(plugins, sp.Name)
		set[sp.Name] = struct{}{}
	}

	compiled := compileExecChain(validated, registry, pc.ConfigVersion())

	var compiledHandlers []gin.HandlerFunc
	if compiled != nil {
		compiledHandlers = make([]gin.HandlerFunc, 0, len(validated))
		for _, sp := range validated {
			plugin, ok := registry.Get(sp.Name)
			if !ok {
				continue
			}

			if middlewarePlugin, ok := plugin.(MiddlewarePlugin); ok && middlewarePlugin.Handler() != nil {
				legacy := middlewarePlugin.Handler()
				compiledHandlers = append(compiledHandlers, func(c *gin.Context) {
					ec := http_proxy_plugin.NewExecContext(c)
					defer http_proxy_plugin.ReleaseExecContext(ec)
					ec.PlanVersion = pc.ConfigVersion()
					legacy(c)
				})
				continue
			}

			compiledHandlers = append(compiledHandlers, func(c *gin.Context) {
				ec := http_proxy_plugin.NewExecContext(c)
				defer http_proxy_plugin.ReleaseExecContext(ec)
				ec.PlanVersion = pc.ConfigVersion()

				result := plugin.Execute(ec)
				if result.IsAbort() {
					handlePluginAbort(c, result)
					return
				}
			})
		}
	}

	return &Plan{
		ServiceID:        serviceID,
		ServiceName:      serviceName,
		ConfigVersion:    pc.ConfigVersion(),
		Plugins:          plugins,
		Warnings:         warnings,
		pluginSet:        set,
		CompiledHandler:  compiled,
		CompiledHandlers: compiledHandlers,
	}
}

func applyPriorityOverrides(specs []PluginSpec, overrides map[string]int) []PluginSpec {
	copied := make([]PluginSpec, len(specs))
	copy(copied, specs)
	if len(overrides) == 0 {
		return copied
	}

	for i := range copied {
		if v, ok := overrides[copied[i].Name]; ok {
			copied[i].Priority = v
		}
	}
	return copied
}

func validateDependencies(enabled []PluginSpec, strict bool) ([]PluginSpec, []string) {
	warnings := make([]string, 0)
	if len(enabled) == 0 {
		return enabled, warnings
	}

	if !strict {
		set := toPluginSet(enabled)
		for _, sp := range enabled {
			for _, dep := range sp.Requires {
				if _, ok := set[dep]; !ok {
					warnings = append(warnings, fmt.Sprintf("plugin %s requires %s but dependency is missing", sp.Name, dep))
				}
			}
		}
		return enabled, warnings
	}

	current := enabled
	for {
		set := toPluginSet(current)
		filtered := make([]PluginSpec, 0, len(current))
		removed := false

		for _, sp := range current {
			missing := ""
			for _, dep := range sp.Requires {
				if _, ok := set[dep]; !ok {
					missing = dep
					break
				}
			}

			if missing != "" {
				removed = true
				warnings = append(warnings, fmt.Sprintf("plugin %s disabled due to missing dependency %s", sp.Name, missing))
				continue
			}

			filtered = append(filtered, sp)
		}

		current = filtered
		if !removed {
			break
		}
	}

	return current, warnings
}

func compileExecChain(specs []PluginSpec, registry *http_proxy_plugin.Registry, planVersion string) gin.HandlerFunc {
	var handlers []gin.HandlerFunc

	for _, sp := range specs {
		plugin, ok := registry.Get(sp.Name)
		if !ok {
			continue
		}

		if middlewarePlugin, ok := plugin.(MiddlewarePlugin); ok && middlewarePlugin.Handler() != nil {
			legacy := middlewarePlugin.Handler()
			handlers = append(handlers, func(c *gin.Context) {
				ec := http_proxy_plugin.NewExecContext(c)
				defer http_proxy_plugin.ReleaseExecContext(ec)
				ec.PlanVersion = planVersion
				legacy(c)
			})
			continue
		}

		handlers = append(handlers, func(c *gin.Context) {
			ec := http_proxy_plugin.NewExecContext(c)
			defer http_proxy_plugin.ReleaseExecContext(ec)
			ec.PlanVersion = planVersion

			result := plugin.Execute(ec)
			if result.IsAbort() {
				handlePluginAbort(c, result)
				return
			}
		})
	}

	return func(c *gin.Context) {
		for _, h := range handlers {
			h(c)
			if c.IsAborted() {
				return
			}
		}
		c.Next()
	}
}

func toPluginSet(specs []PluginSpec) map[string]struct{} {
	set := make(map[string]struct{}, len(specs))
	for _, sp := range specs {
		set[sp.Name] = struct{}{}
	}
	return set
}
