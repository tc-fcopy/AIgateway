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
		{Name: PluginCoreWhiteList, Phase: PhaseEdgeGuard, Priority: 700, Enabled: alwaysOn},
		{Name: PluginCoreBlackList, Phase: PhaseEdgeGuard, Priority: 600, Enabled: alwaysOn},

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
	// ========== 步骤1：应用优先级覆盖规则（服务级优先级 > 插件默认优先级） ==========
	effective := applyPriorityOverrides(specs, pc.PriorityOverrides)
	// 作用：
	// - 入参specs是全局插件规格（默认优先级）；
	// - pc.PriorityOverrides是当前服务的优先级覆盖规则；
	// - 最终effective是「覆盖后」的插件规格列表，保证服务能自定义插件优先级。

	// ========== 步骤2：筛选当前服务需要启用的插件 ==========
	enabled := make([]PluginSpec, 0, len(effective))
	for _, sp := range effective {
		// 核心判断：插件是否为当前服务启用
		// - sp.Enabled == nil：插件无自定义启用规则，默认启用；
		// - sp.Enabled(pc)：执行插件的启用规则（基于PlanContext中的服务配置，如EnableAuth）；
		if sp.Enabled == nil || sp.Enabled(pc) {
			enabled = append(enabled, sp)
		}
	}

	// ========== 步骤3：按「执行阶段+优先级」稳定排序（核心） ==========
	sort.SliceStable(enabled, func(i, j int) bool {
		// 第一优先级：执行阶段（Phase）—— 数值越小，执行越早（如前置阶段<核心阶段）
		if enabled[i].Phase != enabled[j].Phase {
			return enabled[i].Phase < enabled[j].Phase
		}
		// 第二优先级：同阶段内的优先级（Priority）—— 数值越大，执行越早（如auth插件优先级高于log插件）
		return enabled[i].Priority > enabled[j].Priority
	})
	// 关键：sort.SliceStable 保证「同优先级插件」的相对顺序不变（稳定排序），避免排序结果不可预期。

	// ========== 步骤4：校验插件依赖，收集警告 ==========
	validated, warnings := validateDependencies(enabled, pc.StrictDependency)
	// 核心逻辑：
	// - 遍历enabled插件列表，检查每个插件的Dependencies是否都已启用；
	// - pc.StrictDependency=true：依赖缺失则剔除该插件；
	// - pc.StrictDependency=false：依赖缺失仅收集警告，不剔除插件；
	// - 返回validated（校验后的插件列表）和warnings（依赖警告）。

	// ========== 步骤5：提取插件名称，构建快速查询集合 ==========
	plugins := make([]string, 0, len(validated))
	set := make(map[string]struct{}, len(validated))
	for _, sp := range validated {
		plugins = append(plugins, sp.Name) // 最终执行的插件名称列表
		set[sp.Name] = struct{}{}          // 插件集合，用于后续快速判断（如plan.Has(pluginName)）
	}

	// ========== 步骤6：编译执行链 ==========
	compiled := compileExecChain(validated, registry, pc.ConfigVersion())

	// 保存单个 handlers 用于调试
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
					ec.PlanVersion = pc.ConfigVersion()
					legacy(c)
				})
			} else {
				compiledHandlers = append(compiledHandlers, func(c *gin.Context) {
					ec := http_proxy_plugin.NewExecContext(c)
					ec.PlanVersion = pc.ConfigVersion()

					result := plugin.Execute(ec)
					if result.IsAbort() {
						handlePluginAbort(c, result)
						return
					}
				})
			}
		}
	}

	// ========== 步骤7：构建并返回最终的Plan ==========
	return &Plan{
		ServiceID:         serviceID,          // 服务ID
		ServiceName:       serviceName,        // 服务名称
		ConfigVersion:     pc.ConfigVersion(), // 配置版本（用于缓存Key）
		Plugins:           plugins,            // 排序后的插件名称列表（Executor执行的核心依据）
		Warnings:          warnings,           // 构建过程中的警告（如依赖缺失）
		pluginSet:         set,                // 插件集合（优化后续Has方法的查询性能）
		CompiledHandler:   compiled,           // 预编译的执行链
		CompiledHandlers:  compiledHandlers,   // 调试用：单独的 handler 链
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
				ec.PlanVersion = planVersion
				legacy(c)
			})
		} else {
			handlers = append(handlers, func(c *gin.Context) {
				ec := http_proxy_plugin.NewExecContext(c)
				ec.PlanVersion = planVersion

				result := plugin.Execute(ec)
				if result.IsAbort() {
					// 调用 executor.go 中的 handlePluginAbort
					handlePluginAbort(c, result)
					return
				}
			})
		}
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
