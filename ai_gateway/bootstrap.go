package ai_gateway

import (
	"fmt"

	"gateway/ai_gateway/common"
	"gateway/ai_gateway/config"
	"gateway/ai_gateway/consumer"
	"gateway/ai_gateway/observability"
	"gateway/dao"
	"gateway/golang_common/lib"
)

// Bootstrap initializes AI gateway config, runtime components and cache.
func Bootstrap() error {
	if err := config.AIConfManager.LoadOnce(); err != nil {
		return fmt.Errorf("load ai config failed: %w", err)
	}

	conf := config.AIConfManager.GetConfig()
	if conf == nil || !conf.Enable {
		return nil
	}

	redisClient, err := common.NewRedisClient(
		conf.Redis.Addr,
		conf.Redis.Password,
		conf.Redis.DB,
		conf.Redis.PoolSize,
		conf.Redis.Timeout,
	)
	if err != nil {
		return fmt.Errorf("init ai redis failed: %w", err)
	}

	if err := Init(redisClient, lib.GetGormPool); err != nil {
		return fmt.Errorf("init ai components failed: %w", err)
	}

	// Bind runtime module config.
	if GlobalQuotaManager != nil {
		GlobalQuotaManager.SetConfig(
			conf.Enable && conf.DefaultService.EnableQuota,
			conf.Quota.DefaultQuota,
			conf.Quota.QuotaTTL,
		)
	}

	if GlobalPromptDecorator != nil {
		GlobalPromptDecorator.SetConfig(
			conf.Enable && conf.DefaultService.EnablePromptDecorator,
			conf.PromptDecorator.SystemPrefix,
			conf.PromptDecorator.SystemSuffix,
			conf.PromptDecorator.UserPrefix,
			conf.PromptDecorator.UserSuffix,
		)
	}

	if GlobalIPRestrictionManager != nil {
		GlobalIPRestrictionManager.SetGlobalRules(
			conf.IPRestriction.EnableCIDR,
			conf.IPRestriction.Whitelist,
			conf.IPRestriction.Blacklist,
		)
	}

	if conf.Observability.EnableMetrics {
		observability.StartMetricsEndpoint(conf.Observability.MetricsPort)
	}

	// Load consumers from db to in-memory manager.
	rows, err := dao.LoadAIConsumers()
	if err != nil {
		return fmt.Errorf("load ai consumers failed: %w", err)
	}

	list := make([]*consumer.Consumer, 0, len(rows))
	for _, row := range rows {
		item := &consumer.Consumer{
			ID:         row.ID,
			Name:       row.Name,
			Credential: row.Credential,
			Type:       row.Type,
			Status:     row.Status,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		list = append(list, item)
	}
	consumer.ConsumerManager.LoadConsumers(list)

	return nil
}
