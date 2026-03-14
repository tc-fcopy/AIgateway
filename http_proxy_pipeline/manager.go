package http_proxy_pipeline

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/golang_common/lib"
	"gateway/http_proxy_plugin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	defaultPlanner              = NewPlanner(http_proxy_plugin.GlobalRegistry)
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

// GetPlanByServiceID returns latest cached plan for a service.
func GetPlanByServiceID(serviceID int64) (*Plan, bool) {
	return defaultPlanner.GetByServiceID(serviceID)
}

// CachedPlans returns current in-memory plan cache snapshots for admin diagnostics.
func CachedPlans() []CachedPlanSnapshot {
	return defaultPlanner.CachedPlans()
}

// PrebuildPlanForService compiles plan for a service and stores it in cache.
func PrebuildPlanForService(service *dao.ServiceDetail) (*Plan, error) {
	return defaultPlanner.Build(nil, service)
}

// PrebuildPlans compiles plans for a list of services.
func PrebuildPlans(services []*dao.ServiceDetail) error {
	var firstErr error
	for _, service := range services {
		if service == nil || service.Info == nil {
			continue
		}
		if _, err := defaultPlanner.Build(nil, service); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Planner compiles and caches per-service plans.
type Planner struct {
	specs    []PluginSpec
	registry *http_proxy_plugin.Registry // 閹绘帊娆㈠▔銊ュ斀鐞?
	mu       sync.RWMutex
	cache    map[string]*Plan
	latest   map[int64]*Plan
	inflight map[string]*planCall
	buildFn  func(serviceID int64, serviceName string, pc *PlanContext, specs []PluginSpec, registry *http_proxy_plugin.Registry) *Plan
}

func NewPlanner(registry *http_proxy_plugin.Registry) *Planner {
	if registry == nil {
		registry = http_proxy_plugin.GlobalRegistry
	}

	return &Planner{
		specs:    defaultPluginSpecs(),
		registry: registry,
		cache:    map[string]*Plan{},
		latest:   map[int64]*Plan{},
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
	delete(p.latest, serviceID)
}

func (p *Planner) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = map[string]*Plan{}
	p.latest = map[int64]*Plan{}
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

func (p *Planner) GetByServiceID(serviceID int64) (*Plan, bool) {
	if p == nil || serviceID <= 0 {
		return nil, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	plan, ok := p.latest[serviceID]
	return plan, ok && plan != nil
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
		return plan, nil // 缂傛挸鐡ㄩ崨鎴掕厬閿涙氨娲块幒銉ㄧ箲閸ョ儑绱濋弮鐘绘付閸氬海鐢婚幙宥勭稊
	}
	p.mu.RUnlock()

	// ===== 缁楊兛绨╅柌宥嗩梾閺屻儻绱伴崘娆撴敚閸愬秵顐奸弻銉х处鐎涙﹫绱欑憴锝呭枀楠炶泛褰傜粩鐐粹偓渚婄礆=====
	p.mu.Lock()
	if plan, ok := p.cache[cacheKey]; ok {
		p.mu.Unlock()
		return plan, nil // 閸欘垵鍏橀崷銊╁櫞閺€鎹愵嚢闁夸礁鎮楅妴浣稿閸愭瑩鏀ｉ崜宥忕礉瀹稿弶婀侀崗鏈电铂鐠囬攱鐪伴弸鍕紦鐎瑰本鍨?
	}

	// ===== 楠炶泛褰傜拠閿嬬湴閸氬牆鑻熼敍姘梾閺屻儲妲搁崥锔芥箒鐠囬攱鐪板锝呮躬閺嬪嫬缂撶拠顧秎an =====
	if call, ok := p.inflight[cacheKey]; ok {
		// 鐎涙ê婀锝呮躬閺嬪嫬缂撻惃鍕嚞濮瑰偊绱伴柌濠冩杹闁夸緤绱濈粵澶婄窡閺嬪嫬缂撶€瑰本鍨?
		p.mu.Unlock()
		<-call.done                // 闂冭顢ｉ敍宀€娲块崚鐗堢€鍝勭暚閹存劗娈戞穱鈥冲娇
		return call.plan, call.err // 閻╁瓨甯存担璺ㄦ暏瀹稿弶鐎铏规畱plan
	}

	// ===== 閺冪姷绱︾€涙ǜ鈧焦妫ゅ锝呮躬閺嬪嫬缂撻惃鍕嚞濮瑰偊绱板鈧慨瀣€?=====
	// 1. 閸掓稑缂損lanCall閿涘本鐖ｇ拋鎷岊嚉cacheKey濮濓絽婀弸鍕紦
	call := &planCall{done: make(chan struct{})}
	p.inflight[cacheKey] = call // 鐎涙ê鍙唅nflight閿涘牊顒滈崷銊︾€鐚寸礆閺勭姴鐨?
	p.mu.Unlock()               // 闁插﹥鏂侀崘娆撴敚閿涘矂浼╅崗宥夋▎婵夌偛鍙炬禒鏍嚞濮瑰倻娈戝Λ鈧弻銉┾偓鏄忕帆

	// 2. 閹笛嗩攽閻喐顒滈惃鍒緇an閺嬪嫬缂撻柅鏄忕帆閿涘牊鐗宠箛鍐偓妤佹閹垮秳缍旈敍?
	buildFn := p.buildFn
	if buildFn == nil {
		buildFn = buildPlan // 姒涙顓婚弸鍕紦閸戣姤鏆?
	}
	plan := buildFn(pc.ServiceID, pc.ServiceName, pc, p.specs, p.registry)

	// ===== 閺嬪嫬缂撶€瑰本鍨氶敍姘纯閺傛壆绱︾€?+ 闁氨鐓＄粵澶婄窡閻ㄥ嫯顕Ч?=====
	p.mu.Lock()
	// 缁夊娅巌nflight閺嶅洩顔囬敍鍫熺€鍝勭暚閹存劧绱?
	delete(p.inflight, cacheKey)
	// 濞撳懐鎮婄拠銉︽箛閸旓紕娈戦弮褏澧楅張鐞緇an缂傛挸鐡ㄩ敍鍫ヤ缉閸忓秴鍞寸€涙ɑ纭犲蹇ョ礆
	prefix := fmt.Sprintf("%d:", pc.ServiceID)
	for k := range p.cache {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix && k != cacheKey {
			delete(p.cache, k)
		}
	}
	// 鐏忓棙鏌妏lan鐎涙ê鍙嗙紓鎾崇摠
	p.cache[cacheKey] = plan
	if plan != nil {
		p.latest[pc.ServiceID] = plan
	}
	// 婵夘偄鍘杙lanCall缂佹挻鐏夐敍宀勨偓姘辩叀缁涘绶熼惃鍕嚞濮?
	call.plan = plan
	close(call.done) // 閸忔娊妫撮柅姘朵壕閿涘本澧嶉張澶岀搼瀵板懐娈戠拠閿嬬湴娴兼俺顫﹂崬銈夊晪
	p.mu.Unlock()

	// 鏉╂柨娲栭弸鍕紦婵傜晫娈憄lan
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
