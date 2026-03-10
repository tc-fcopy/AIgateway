package http_proxy_pipeline

import (
	"testing"

	"gateway/http_proxy_plugin"
)

func TestBuildPlan_StrictDependencyDisablesPlugin(t *testing.T) {
	pc := &PlanContext{
		AIEnabled:         true,
		EnableAuth:        false,
		EnableTokenLimit:  true,
		StrictDependency:  true,
		PriorityOverrides: map[string]int{},
	}

	plan := buildPlan(1, "svc", pc, defaultPluginSpecs(), http_proxy_plugin.GlobalRegistry)
	if plan.Has(PluginAITokenRateLimit) {
		t.Fatalf("expected %s to be disabled when %s is missing in strict mode", PluginAITokenRateLimit, PluginAIAuth)
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected dependency warning, got none")
	}
}

func TestBuildPlan_NonStrictDependencyKeepsPlugin(t *testing.T) {
	pc := &PlanContext{
		AIEnabled:         true,
		EnableAuth:        false,
		EnableTokenLimit:  true,
		StrictDependency:  false,
		PriorityOverrides: map[string]int{},
	}

	plan := buildPlan(1, "svc", pc, defaultPluginSpecs(), http_proxy_plugin.GlobalRegistry)
	if !plan.Has(PluginAITokenRateLimit) {
		t.Fatalf("expected %s to remain enabled in non-strict mode", PluginAITokenRateLimit)
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected dependency warning, got none")
	}
}

func TestBuildPlan_PriorityOverride(t *testing.T) {
	pc := &PlanContext{
		AIEnabled:         true,
		EnableAuth:        true,
		EnableTokenLimit:  true,
		EnableQuota:       true,
		StrictDependency:  true,
		PriorityOverrides: map[string]int{PluginAIQuota: 990},
	}

	plan := buildPlan(1, "svc", pc, defaultPluginSpecs(), http_proxy_plugin.GlobalRegistry)
	quotaIdx := indexOf(plan.Plugins, PluginAIQuota)
	rateIdx := indexOf(plan.Plugins, PluginAITokenRateLimit)
	if quotaIdx < 0 || rateIdx < 0 {
		t.Fatalf("expected both quota and token_ratelimit plugins in plan")
	}
	if quotaIdx > rateIdx {
		t.Fatalf("expected quota plugin to run before token_ratelimit after override, quotaIdx=%d rateIdx=%d", quotaIdx, rateIdx)
	}
}

func indexOf(list []string, target string) int {
	for i, s := range list {
		if s == target {
			return i
		}
	}
	return -1
}

func TestPlannerCachedPlans_ReturnsCopy(t *testing.T) {
	p := NewPlanner(nil)
	p.cache["1:test"] = &Plan{
		ServiceID:     1,
		ServiceName:   "svc-test",
		ConfigVersion: "v1",
		Plugins:       []string{PluginCoreFlowCount, PluginProxyReverseProxy},
		Warnings:      []string{"warn"},
		pluginSet:     map[string]struct{}{PluginCoreFlowCount: {}, PluginProxyReverseProxy: {}},
	}

	snapshots := p.CachedPlans()
	if len(snapshots) != 1 {
		t.Fatalf("expected one cached plan snapshot, got %d", len(snapshots))
	}
	if snapshots[0].CacheKey != "1:test" {
		t.Fatalf("unexpected cache key: %s", snapshots[0].CacheKey)
	}
	if len(snapshots[0].Plugins) != 2 {
		t.Fatalf("unexpected plugin count: %d", len(snapshots[0].Plugins))
	}

	snapshots[0].Plugins[0] = "changed"
	if p.cache["1:test"].Plugins[0] == "changed" {
		t.Fatalf("expected snapshot plugins to be copied")
	}
}
