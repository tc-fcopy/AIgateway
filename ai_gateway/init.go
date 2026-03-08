package ai_gateway

import (
	"sync"

	"gateway/ai_gateway/cache"
	"gateway/ai_gateway/common"
	"gateway/ai_gateway/loadbalancer"
	"gateway/ai_gateway/model"
	"gateway/ai_gateway/prompt"
	"gateway/ai_gateway/quota"
	"gateway/ai_gateway/ratelimit"
	"gateway/ai_gateway/security"
	"gorm.io/gorm"
)

var (
	GlobalRedisClient          *common.RedisClient
	GlobalTokenLimiter         *ratelimit.TokenLimiter
	GlobalQuotaManager         *quota.Manager
	GlobalModelRouter          *model.ModelRouter
	GlobalModelMapper          *model.ModelMapper
	GlobalStringCache          *cache.StringCache
	GlobalLoadBalancer         *loadbalancer.GlobalLeastRequest
	GlobalPromptDecorator      *prompt.PromptDecorator
	GlobalIPRestrictionManager *security.IPRestrictionManager

	once        sync.Once
	initialized bool
)

func Init(redisClient *common.RedisClient, getDBPool func(string) (*gorm.DB, error)) error {
	once.Do(func() {
		GlobalRedisClient = redisClient
		GlobalTokenLimiter = ratelimit.NewTokenLimiter(redisClient, true)
		GlobalQuotaManager = quota.NewManager()
		GlobalQuotaManager.Init(redisClient, nil, getDBPool)

		GlobalModelRouter = model.GlobalModelRouter
		GlobalModelMapper = model.GlobalModelMapper
		GlobalStringCache = cache.NewStringCache(redisClient)
		GlobalLoadBalancer = loadbalancer.NewGlobalLeastRequest()
		GlobalPromptDecorator = prompt.NewPromptDecorator()
		GlobalIPRestrictionManager = security.NewIPRestrictionManager()

		initialized = true
	})

	return nil
}

func IsInitialized() bool {
	return initialized
}

func GetRedisClient() *common.RedisClient {
	return GlobalRedisClient
}

func GetTokenLimiter() *ratelimit.TokenLimiter {
	return GlobalTokenLimiter
}

func GetQuotaManager() *quota.Manager {
	return GlobalQuotaManager
}

func GetModelRouter() *model.ModelRouter {
	return GlobalModelRouter
}

func GetModelMapper() *model.ModelMapper {
	return GlobalModelMapper
}

func GetStringCache() *cache.StringCache {
	return GlobalStringCache
}

func GetLoadBalancer() *loadbalancer.GlobalLeastRequest {
	return GlobalLoadBalancer
}

func GetPromptDecorator() *prompt.PromptDecorator {
	return GlobalPromptDecorator
}

func GetIPRestrictionManager() *security.IPRestrictionManager {
	return GlobalIPRestrictionManager
}
