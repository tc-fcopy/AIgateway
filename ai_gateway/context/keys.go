package context

// Context Key 定义
const (
	// ConsumerKey 存储Consumer对象
	ConsumerKey = "ai_consumer"
	// ConsumerNameKey 存储Consumer名称
	ConsumerNameKey = "ai_consumer_name"
	// JWTClaimsKey 存储JWT Claims
	JWTClaimsKey = "ai_jwt_claims"
	// TokenUsageKey 存储Token使用信息
	TokenUsageKey = "ai_token_usage"
	// RateLimitKey 存储限流配置
	RateLimitKey = "ai_rate_limit"
	// RateLimitConfigKey 存储限流配置对象
	RateLimitConfigKey = "ai_rate_limit_config"
	// ModelKey 存储模型名称
	ModelKey = "ai_model"
	// OriginalModelKey 存储原始模型名称
	OriginalModelKey = "ai_original_model"
	// CacheKey 存储缓存Key
	CacheKey = "ai_cache_key"
	// SkipCacheKey 跳过缓存标志
	SkipCacheKey = "ai_skip_cache"
	// BackendIDKey 存储后端ID
	BackendIDKey = "ai_backend_id"
	// BackendAddressKey 存储后端地址
	BackendAddressKey = "ai_backend_address"
	// StartTimeKey 请求开始时间
	StartTimeKey = "ai_start_time"
	// FirstTokenTimeKey 首字时间
	FirstTokenTimeKey = "ai_first_token_time"
)
