package http_proxy_pipeline

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/golang_common/lib"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	defaultPlanner              = NewPlanner()
	aiServiceConfigRuntimeCache = newAIServiceConfigRuntime()
	aiServiceConfigBulkLoader   = defaultAIServiceConfigBulkLoader
	aiServiceConfigSingleLoader = defaultAIServiceConfigSingleLoader
)

type planCall struct {
	done chan struct{}
	plan *Plan
	err  error
}

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
	specs    []PluginSpec
	mu       sync.RWMutex
	cache    map[string]*Plan
	inflight map[string]*planCall
	buildFn  func(serviceID int64, serviceName string, pc *PlanContext, specs []PluginSpec) *Plan
}

func NewPlanner() *Planner {
	return &Planner{
		specs:    defaultPluginSpecs(),
		cache:    map[string]*Plan{},
		inflight: map[string]*planCall{},
		buildFn:  buildPlan,
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
	// 前置校验：服务配置不能为空
	if service == nil || service.Info == nil {
		return nil, fmt.Errorf("service detail is nil")
	}

	// 1. 构建计划上下文（封装服务ID、配置版本、插件配置等）
	pc, err := BuildPlanContext(c, service)
	if err != nil {
		return nil, err
	}

	// 2. 生成全局唯一的缓存Key：服务ID + 配置版本（核心标识）
	cacheKey := fmt.Sprintf("%d:%s", pc.ServiceID, pc.ConfigVersion())

	// ===== 第一重检查：读锁查缓存（无阻塞，高性能）=====
	p.mu.RLock()
	if plan, ok := p.cache[cacheKey]; ok {
		p.mu.RUnlock()
		return plan, nil // 缓存命中：直接返回，无需后续操作
	}
	p.mu.RUnlock()

	// ===== 第二重检查：写锁再次查缓存（解决并发竞态）=====
	p.mu.Lock()
	if plan, ok := p.cache[cacheKey]; ok {
		p.mu.Unlock()
		return plan, nil // 可能在释放读锁后、加写锁前，已有其他请求构建完成
	}

	// ===== 并发请求合并：检查是否有请求正在构建该plan =====
	if call, ok := p.inflight[cacheKey]; ok {
		// 存在正在构建的请求：释放锁，等待构建完成
		p.mu.Unlock()
		<-call.done                // 阻塞，直到构建完成的信号
		return call.plan, call.err // 直接使用已构建的plan
	}

	// ===== 无缓存、无正在构建的请求：开始构建 =====
	// 1. 创建planCall，标记该cacheKey正在构建
	call := &planCall{done: make(chan struct{})}
	p.inflight[cacheKey] = call // 存入inflight（正在构建）映射
	p.mu.Unlock()               // 释放写锁，避免阻塞其他请求的检查逻辑

	// 2. 执行真正的plan构建逻辑（核心耗时操作）
	buildFn := p.buildFn
	if buildFn == nil {
		buildFn = buildPlan // 默认构建函数
	}
	plan := buildFn(pc.ServiceID, pc.ServiceName, pc, p.specs)

	// ===== 构建完成：更新缓存 + 通知等待的请求 =====
	p.mu.Lock()
	// 移除inflight标记（构建完成）
	delete(p.inflight, cacheKey)
	// 清理该服务的旧版本plan缓存（避免内存泄漏）
	prefix := fmt.Sprintf("%d:", pc.ServiceID)
	for k := range p.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix && k != cacheKey {
			delete(p.cache, k)
		}
	}
	// 将新plan存入缓存
	p.cache[cacheKey] = plan
	// 填充planCall结果，通知等待的请求
	call.plan = plan
	close(call.done) // 关闭通道，所有等待的请求会被唤醒
	p.mu.Unlock()

	// 返回构建好的plan
	return plan, nil
}

// BuildPlanContext builds a service-aware capability context.
func BuildPlanContext(c *gin.Context, service *dao.ServiceDetail) (*PlanContext, error) {
	// 服务专属配置 > 全局服务优先级覆盖规则 > 全局默认配置
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
	// Read from runtime cache to avoid per-request DB query.
	if pc.ServiceID > 0 {
		row, ok, err := aiServiceConfigRuntimeCache.Get(pc.ServiceID)
		if err == nil && ok && row != nil {
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

	return pc, nil
}

type aiServiceConfigRuntime struct {
	mu     sync.RWMutex
	cache  map[int64]*dao.AIServiceConfig
	loaded bool
}

func newAIServiceConfigRuntime() *aiServiceConfigRuntime {
	return &aiServiceConfigRuntime{
		cache: map[int64]*dao.AIServiceConfig{},
	}
}

// ReloadAIServiceConfigRuntime refreshes AI service config runtime cache.
// serviceID > 0 refreshes one service; otherwise reloads all.
func ReloadAIServiceConfigRuntime(serviceID int64) error {
	return aiServiceConfigRuntimeCache.Reload(serviceID)
}

func (r *aiServiceConfigRuntime) Reload(serviceID int64) error {
	if serviceID > 0 {
		return r.reloadOne(serviceID)
	}
	return r.reloadAll()
}

func (r *aiServiceConfigRuntime) Get(serviceID int64) (*dao.AIServiceConfig, bool, error) {
	if serviceID <= 0 {
		return nil, false, nil
	}
	if err := r.ensureLoaded(); err != nil {
		return nil, false, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	row, ok := r.cache[serviceID]
	if !ok || row == nil {
		return nil, false, nil
	}
	copied := *row
	return &copied, true, nil
}

func (r *aiServiceConfigRuntime) ensureLoaded() error {
	r.mu.RLock()
	loaded := r.loaded
	r.mu.RUnlock()
	if loaded {
		return nil
	}
	return r.reloadAll()
}

func (r *aiServiceConfigRuntime) reloadAll() error {
	rows, err := aiServiceConfigBulkLoader()
	if err != nil {
		return err
	}

	next := make(map[int64]*dao.AIServiceConfig, len(rows))
	for i := range rows {
		row := rows[i]
		copied := row
		next[row.ServiceID] = &copied
	}

	r.mu.Lock()
	r.cache = next
	r.loaded = true
	r.mu.Unlock()
	return nil
}

func (r *aiServiceConfigRuntime) reloadOne(serviceID int64) error {
	row, err := aiServiceConfigSingleLoader(serviceID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if row == nil {
		delete(r.cache, serviceID)
		return nil
	}

	copied := *row
	r.cache[serviceID] = &copied
	return nil
}

func defaultAIServiceConfigBulkLoader() ([]dao.AIServiceConfig, error) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return nil, err
	}

	rows := make([]dao.AIServiceConfig, 0)
	if err := tx.WithContext(context.Background()).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func defaultAIServiceConfigSingleLoader(serviceID int64) (*dao.AIServiceConfig, error) {
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return nil, err
	}

	row := &dao.AIServiceConfig{}
	err = tx.WithContext(context.Background()).Where("service_id = ?", serviceID).First(row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func resetAIServiceConfigRuntimeForTest() {
	aiServiceConfigRuntimeCache.mu.Lock()
	aiServiceConfigRuntimeCache.cache = map[int64]*dao.AIServiceConfig{}
	aiServiceConfigRuntimeCache.loaded = false
	aiServiceConfigRuntimeCache.mu.Unlock()
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
