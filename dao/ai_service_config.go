package dao

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

// AIServiceConfig AI服务配置
type AIServiceConfig struct {
	ID                    int64     `json:"id" gorm:"primary_key;column:id"`
	ServiceID             int64     `json:"service_id" gorm:"column:service_id;description:服务ID"`
	EnableKeyAuth         bool      `json:"enable_key_auth" gorm:"column:enable_key_auth;description:是否开启Key Auth"`
	EnableJWTAuth         bool      `json:"enable_jwt_auth" gorm:"column:enable_jwt_auth;description:是否开启JWT Auth"`
	EnableTokenRateLimit  bool      `json:"enable_token_ratelimit" gorm:"column:enable_token_ratelimit;description:是否开启Token限流"`
	EnableQuota           bool      `json:"enable_quota" gorm:"column:enable_quota;description:是否开启配额"`
	EnableModelRouter      bool      `json:"enable_model_router" gorm:"column:enable_model_router;description:是否开启模型路由"`
	EnableModelMapper      bool      `json:"enable_model_mapper" gorm:"column:enable_model_mapper;description:是否开启模型映射"`
	EnableCache           bool      `json:"enable_cache" gorm:"column:enable_cache;description:是否开启缓存"`
	EnableLoadBalancer     bool      `json:"enable_loadbalancer" gorm:"column:enable_loadbalancer;description:是否开启负载均衡"`
	EnableObservability    bool      `json:"enable_observability" gorm:"column:enable_observability;description:是否开启可观测性"`
	EnablePromptDecorator  bool      `json:"enable_prompt_decorator" gorm:"column:enable_prompt_decorator;description:是否开启Prompt装饰"`
	EnableIPRestriction    bool      `json:"enable_ip_restriction" gorm:"column:enable_ip_restriction;description:是否开启IP限制"`
	EnableCORS            bool      `json:"enable_cors" gorm:"column:enable_cors;description:是否开启CORS"`
	CreatedAt             time.Time `json:"created_at" gorm:"column:create_at;description:创建时间"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"column:update_at;description:更新时间"`
}

// TableName 返回表名
func (a *AIServiceConfig) TableName() string {
	return "ai_service_config"
}

// Find 查询单个AI服务配置
func (a *AIServiceConfig) Find(c *gin.Context, tx *gorm.DB, search *AIServiceConfig) (*AIServiceConfig, error) {
	out := &AIServiceConfig{}
	err := tx.WithContext(c).Where(search).First(out).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

// Save 保存AI服务配置
func (a *AIServiceConfig) Save(c *gin.Context, tx *gorm.DB) error {
	return tx.WithContext(c).Save(a).Error
}

// GetByServiceID 根据服务ID获取AI配置
func (a *AIServiceConfig) GetByServiceID(c *gin.Context, tx *gorm.DB, serviceID int64) (*AIServiceConfig, error) {
	return a.Find(c, tx, &AIServiceConfig{ServiceID: serviceID})
}
