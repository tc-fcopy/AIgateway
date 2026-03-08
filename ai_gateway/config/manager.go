package config

import (
	"sync"

	"gateway/ai_gateway/model"
	"gateway/golang_common/lib"
)

var (
	AIConf        *AIConfig
	AIConfManager *ConfigManager
)

func init() {
	AIConfManager = NewConfigManager()
}

// ConfigManager manages global AI config.
type ConfigManager struct {
	conf *AIConfig
	lock sync.RWMutex
	once sync.Once
	err  error
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		conf: &AIConfig{},
	}
}

func (m *ConfigManager) LoadOnce() error {
	m.once.Do(func() {
		conf := &AIConfig{}
		if err := lib.ParseConfig(lib.GetConfPath("ai"), conf); err != nil {
			m.err = err
			return
		}

		m.lock.Lock()
		m.conf = conf
		AIConf = conf
		m.lock.Unlock()

		m.initGlobalComponents()
	})
	return m.err
}

func (m *ConfigManager) SetConfig(conf *AIConfig) {
	m.lock.Lock()
	m.conf = conf
	AIConf = conf
	m.lock.Unlock()

	m.initGlobalComponents()
}

func (m *ConfigManager) GetConfig() *AIConfig {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.conf
}

func (m *ConfigManager) initGlobalComponents() {
	conf := m.GetConfig()
	if conf == nil {
		return
	}

	rules := make([]model.ModelRule, len(conf.ModelRouter.Rules))
	for i, rule := range conf.ModelRouter.Rules {
		rules[i] = model.ModelRule{
			Pattern:     rule.Pattern,
			TargetModel: rule.TargetModel,
			Priority:    len(conf.ModelRouter.Rules) - i,
		}
	}

	model.GlobalModelRouter.SetConfig(
		conf.Enable && conf.DefaultService.EnableModelRouter,
		conf.ModelRouter.DefaultModel,
		rules,
	)

	mappings := make([]model.ModelMapping, len(conf.ModelMapper.Mappings))
	for i, mapping := range conf.ModelMapper.Mappings {
		mappings[i] = model.ModelMapping{
			Source: mapping.Source,
			Target: mapping.Target,
		}
	}
	model.GlobalModelMapper.SetConfig(
		mappings,
		conf.Enable && conf.DefaultService.EnableModelMapper,
	)
}

func (m *ConfigManager) IsEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable
}

func (m *ConfigManager) IsKeyAuthEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableKeyAuth
}

func (m *ConfigManager) IsJWTAuthEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableJWTAuth
}

func (m *ConfigManager) IsTokenRateLimitEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableTokenRateLimit
}

func (m *ConfigManager) IsQuotaEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableQuota
}

func (m *ConfigManager) IsModelRouterEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableModelRouter
}

func (m *ConfigManager) IsModelMapperEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableModelMapper
}

func (m *ConfigManager) IsCacheEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableCache
}

func (m *ConfigManager) IsLoadBalancerEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableLoadBalancer
}

func (m *ConfigManager) IsObservabilityEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableObservability
}

func (m *ConfigManager) IsPromptDecoratorEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnablePromptDecorator
}

func (m *ConfigManager) IsIPRestrictionEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableIPRestriction
}

func (m *ConfigManager) IsCORSEnabled() bool {
	conf := m.GetConfig()
	return conf != nil && conf.Enable && conf.DefaultService.EnableCORS
}

