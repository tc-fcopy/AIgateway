package config

// AIConfig is the root config for AI gateway features.
type AIConfig struct {
	Enable             bool                  `mapstructure:"enable"`
	ApplyToAllServices *bool                 `mapstructure:"apply_to_all_services"`
	DefaultService     AIServiceConfig       `mapstructure:"default_service"`
	KeyAuth            KeyAuthConfig         `mapstructure:"key_auth"`
	JWTAuth            JWTAuthConfig         `mapstructure:"jwt_auth"`
	TokenRateLimit     TokenRateLimitConfig  `mapstructure:"token_ratelimit"`
	Quota              QuotaConfig           `mapstructure:"quota"`
	ModelRouter        ModelRouterConfig     `mapstructure:"model_router"`
	ModelMapper        ModelMapperConfig     `mapstructure:"model_mapper"`
	Cache              CacheConfig           `mapstructure:"cache"`
	LoadBalancer       LoadBalancerConfig    `mapstructure:"loadbalancer"`
	Observability      ObservabilityConfig   `mapstructure:"observability"`
	PromptDecorator    PromptDecoratorConfig `mapstructure:"prompt_decorator"`
	IPRestriction      IPRestrictionConfig   `mapstructure:"ip_restriction"`
	CORS               CORSConfig            `mapstructure:"cors"`
	Redis              RedisConfig           `mapstructure:"redis"`
	Pipeline           PipelineConfig        `mapstructure:"pipeline"`
}

type AIServiceConfig struct {
	EnableKeyAuth         bool `mapstructure:"enable_key_auth"`
	EnableJWTAuth         bool `mapstructure:"enable_jwt_auth"`
	EnableTokenRateLimit  bool `mapstructure:"enable_token_ratelimit"`
	EnableQuota           bool `mapstructure:"enable_quota"`
	EnableModelRouter     bool `mapstructure:"enable_model_router"`
	EnableModelMapper     bool `mapstructure:"enable_model_mapper"`
	EnableCache           bool `mapstructure:"enable_cache"`
	EnableLoadBalancer    bool `mapstructure:"enable_loadbalancer"`
	EnableObservability   bool `mapstructure:"enable_observability"`
	EnablePromptDecorator bool `mapstructure:"enable_prompt_decorator"`
	EnableIPRestriction   bool `mapstructure:"enable_ip_restriction"`
	EnableCORS            bool `mapstructure:"enable_cors"`
}

type KeyAuthConfig struct {
	KeyNames       []string `mapstructure:"key_names"`
	AllowConsumers []string `mapstructure:"allow_consumers"`
}

type JWTAuthConfig struct {
	Secret          string   `mapstructure:"secret"`
	Algorithms      []string `mapstructure:"algorithms"`
	AllowConsumers  []string `mapstructure:"allow_consumers"`
	TokenHeader     string   `mapstructure:"token_header"`
	TokenQueryParam string   `mapstructure:"token_query_param"`
}

type TokenRateLimitConfig struct {
	WindowSeconds int64    `mapstructure:"window_seconds"`
	LimitTokens   int64    `mapstructure:"limit_tokens"`
	Windows       []string `mapstructure:"windows"`
}

type QuotaConfig struct {
	DefaultQuota int64 `mapstructure:"default_quota"`
	QuotaTTL     int64 `mapstructure:"quota_ttl"`
}

type ModelRouterConfig struct {
	EnableAuto   bool        `mapstructure:"enable_auto"`
	DefaultModel string      `mapstructure:"default_model"`
	Rules        []ModelRule `mapstructure:"rules"`
}

type ModelRule struct {
	Pattern     string `mapstructure:"pattern"`
	TargetModel string `mapstructure:"target_model"`
}

type ModelMapperConfig struct {
	Mappings []ModelMapping `mapstructure:"mappings"`
}

type ModelMapping struct {
	Source string `mapstructure:"source"`
	Target string `mapstructure:"target"`
}

type CacheConfig struct {
	CacheTTL     int64 `mapstructure:"cache_ttl"`
	MaxCacheSize int   `mapstructure:"max_cache_size"`
	CacheStream  bool  `mapstructure:"cache_stream"`
}

type LoadBalancerConfig struct {
	Strategy            string `mapstructure:"strategy"`
	HealthCheckInterval int    `mapstructure:"health_check_interval"`
	NodeTimeout         int    `mapstructure:"node_timeout"`
}

type ObservabilityConfig struct {
	EnableMetrics bool   `mapstructure:"enable_metrics"`
	MetricsPort   int    `mapstructure:"metrics_port"`
	EnableLog     bool   `mapstructure:"enable_log"`
	LogLevel      string `mapstructure:"log_level"`
	LogRequest    bool   `mapstructure:"log_request"`
	LogResponse   bool   `mapstructure:"log_response"`
}

type PromptDecoratorConfig struct {
	SystemPrefix string `mapstructure:"system_prefix"`
	SystemSuffix string `mapstructure:"system_suffix"`
	UserPrefix   string `mapstructure:"user_prefix"`
	UserSuffix   string `mapstructure:"user_suffix"`
}

type IPRestrictionConfig struct {
	Whitelist  []string `mapstructure:"whitelist"`
	Blacklist  []string `mapstructure:"blacklist"`
	EnableCIDR bool     `mapstructure:"enable_cidr"`
}

type CORSConfig struct {
	AllowAllOrigins  bool     `mapstructure:"allow_all_origins"`
	AllowedOrigins   []string `mapstructure:"allow_origins"`
	AllowedMethods   []string `mapstructure:"allow_methods"`
	AllowedHeaders   []string `mapstructure:"allow_headers"`
	ExposedHeaders   []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
	Timeout  int    `mapstructure:"timeout"`
}

type PipelineConfig struct {
	StrictDependency         bool                      `mapstructure:"strict_dependency"`
	PriorityOverrides        map[string]int            `mapstructure:"priority_overrides"`
	ServicePriorityOverrides map[string]map[string]int `mapstructure:"service_priority_overrides"`
}
