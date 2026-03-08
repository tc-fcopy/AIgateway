package http_proxy_pipeline

import (
	"fmt"
	"strconv"
	"sync"

	"gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/golang_common/lib"
	"github.com/gin-gonic/gin"
)

var (
	defaultPlanner = NewPlanner()
)

// CachedPlanSnapshot is a read-only view for plan cache diagnostics.
type CachedPlanSnapshot struct {
	CacheKey      string   `json:"cache_key"`
	ServiceID     int64    `json:"service_id"`
	ServiceName   string   `json:"service_name"`
	ConfigVersion string   `json:"config_version"`
	Plugins       []string `json:"plugins"`
	Warnings      []string `json:"warnings"`
}

// InvalidateService clears cached plan versions for one service.
func InvalidateService(serviceID int64) {
	defaultPlanner.Invalidate(serviceID)
}

// InvalidateAll clears all cached plans.
func InvalidateAll() {
	defaultPlanner.InvalidateAll()
}

// BuildPlanForService compiles (or returns cached) plan for a concrete service.
func BuildPlanForService(c *gin.Context, service *dao.ServiceDetail) (*Plan, error) {
	return defaultPlanner.Build(c, service)
}

// CachedPlans returns current in-memory plan cache snapshots for admin diagnostics.
func CachedPlans() []CachedPlanSnapshot {
	return defaultPlanner.CachedPlans()
}

// Planner compiles and caches per-service plans.
type Planner struct {
	specs []PluginSpec
	mu    sync.RWMutex
	cache map[string]*Plan
}

func NewPlanner() *Planner {
	return &Planner{
		specs: defaultPluginSpecs(),
		cache: map[string]*Plan{},
	}
}

func (p *Planner) Invalidate(serviceID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	prefix := fmt.Sprintf("%d:", serviceID)
	for k := range p.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(p.cache, k)
		}
	}
}

func (p *Planner) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = map[string]*Plan{}
}

func (p *Planner) CachedPlans() []CachedPlanSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]CachedPlanSnapshot, 0, len(p.cache))
	for key, plan := range p.cache {
		if plan == nil {
			continue
		}
		plugins := make([]string, len(plan.Plugins))
		copy(plugins, plan.Plugins)
		warnings := make([]string, len(plan.Warnings))
		copy(warnings, plan.Warnings)

		out = append(out, CachedPlanSnapshot{
			CacheKey:      key,
			ServiceID:     plan.ServiceID,
			ServiceName:   plan.ServiceName,
			ConfigVersion: plan.ConfigVersion,
			Plugins:       plugins,
			Warnings:      warnings,
		})
	}
	return out
}

func (p *Planner) Build(c *gin.Context, service *dao.ServiceDetail) (*Plan, error) {
	if service == nil || service.Info == nil {
		return nil, fmt.Errorf("service detail is nil")
	}

	pc, err := BuildPlanContext(c, service)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("%d:%s", pc.ServiceID, pc.ConfigVersion())

	p.mu.RLock()
	if plan, ok := p.cache[cacheKey]; ok {
		p.mu.RUnlock()
		return plan, nil
	}
	p.mu.RUnlock()

	plan := buildPlan(pc.ServiceID, pc.ServiceName, pc, p.specs)

	p.mu.Lock()
	// auto invalidate stale versions for the same service id.
	prefix := fmt.Sprintf("%d:", pc.ServiceID)
	for k := range p.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix && k != cacheKey {
			delete(p.cache, k)
		}
	}
	p.cache[cacheKey] = plan
	p.mu.Unlock()

	return plan, nil
}

// BuildPlanContext builds a service-aware capability context.
func BuildPlanContext(c *gin.Context, service *dao.ServiceDetail) (*PlanContext, error) {
	pc := &PlanContext{}
	if service != nil && service.Info != nil {
		pc.ServiceID = service.Info.ID
		pc.ServiceName = service.Info.ServiceName
		pc.LoadType = service.Info.LoadType
	}

	conf := config.AIConfManager.GetConfig()
	if conf != nil && conf.Enable {
		applyAll := true
		if conf.ApplyToAllServices != nil {
			applyAll = *conf.ApplyToAllServices
		}

		pc.AIEnabled = applyAll
		pc.EnableCORS = conf.DefaultService.EnableCORS
		pc.EnableAuth = conf.DefaultService.EnableKeyAuth || conf.DefaultService.EnableJWTAuth
		pc.EnableIPRestrict = conf.DefaultService.EnableIPRestriction
		pc.EnableModelRouter = conf.DefaultService.EnableModelMapper || conf.DefaultService.EnableModelRouter
		pc.EnablePrompt = conf.DefaultService.EnablePromptDecorator
		pc.EnableTokenLimit = conf.DefaultService.EnableTokenRateLimit
		pc.EnableQuota = conf.DefaultService.EnableQuota
		pc.EnableCache = conf.DefaultService.EnableCache
		pc.EnableLoadBalance = conf.DefaultService.EnableLoadBalancer
		pc.EnableObserve = conf.DefaultService.EnableObservability

		pc.StrictDependency = conf.Pipeline.StrictDependency
		pc.PriorityOverrides = copyPriorityMap(conf.Pipeline.PriorityOverrides)

		if len(conf.Pipeline.ServicePriorityOverrides) > 0 {
			serviceKey := strconv.FormatInt(pc.ServiceID, 10)
			if m, ok := conf.Pipeline.ServicePriorityOverrides[serviceKey]; ok {
				mergePriorityMap(pc.PriorityOverrides, m)
			}
			if m, ok := conf.Pipeline.ServicePriorityOverrides[pc.ServiceName]; ok {
				mergePriorityMap(pc.PriorityOverrides, m)
			}
		}
	}

	// Service-level AI config overrides global defaults when present.
	if pc.ServiceID > 0 {
		tx, err := lib.GetGormPool("default")
		if err == nil {
			model := &dao.AIServiceConfig{}
			row, qErr := model.GetByServiceID(c, tx, pc.ServiceID)
			if qErr == nil && row != nil {
				pc.AIEnabled = true
				pc.EnableAuth = row.EnableKeyAuth || row.EnableJWTAuth
				pc.EnableIPRestrict = row.EnableIPRestriction
				pc.EnableModelRouter = row.EnableModelRouter || row.EnableModelMapper
				pc.EnablePrompt = row.EnablePromptDecorator
				pc.EnableTokenLimit = row.EnableTokenRateLimit
				pc.EnableQuota = row.EnableQuota
				pc.EnableCache = row.EnableCache
				pc.EnableLoadBalance = row.EnableLoadBalancer
				pc.EnableObserve = row.EnableObservability
				pc.EnableCORS = row.EnableCORS
			}
		}
	}

	return pc, nil
}

func copyPriorityMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergePriorityMap(dst map[string]int, src map[string]int) {
	if len(src) == 0 {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}
