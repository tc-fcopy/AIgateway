package quota

import (
	"errors"
	"fmt"

	"gateway/ai_gateway/common"
	ratelimit "gateway/ai_gateway/ratelimit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	// QuotaManager 全局配额管理器
	QuotaManager *Manager
)

func init() {
	QuotaManager = NewManager()
}

// Manager 配额管理器
type Manager struct {
	redisClient *common.RedisClient
	db          *gorm.DB
	enable      bool
	defaultQuota int64
	quotaTTL    int64
	getDBPool   func(string) (*gorm.DB, error) // 数据库连接池获取函数
}

// NewManager 创建配额管理器
func NewManager() *Manager {
	return &Manager{
		enable:      true,
		defaultQuota: 100000,
		quotaTTL:    86400, // 24小时
	}
}

// Init 初始化配额管理器
func (m *Manager) Init(redisClient *common.RedisClient, db *gorm.DB, getDBPool func(string) (*gorm.DB, error)) {
	m.redisClient = redisClient
	m.db = db
	m.getDBPool = getDBPool
}

// SetConfig 设置配置
func (m *Manager) SetConfig(enable bool, defaultQuota, quotaTTL int64) {
	m.enable = enable
	m.defaultQuota = defaultQuota
	m.quotaTTL = quotaTTL
}

// GetQuota 获取Consumer剩余配额
func (m *Manager) GetQuota(c *gin.Context, consumerName string) (int64, error) {
	if !m.enable {
		return 0, nil
	}

	// 优先从Redis获取
	redisKey := common.BuildQuotaKey(consumerName)
	quotaStr, err := m.redisClient.Get(redisKey)
	if err == nil && quotaStr != "" {
		quota := parseInt64(quotaStr)
		return quota, nil
	}

	// Redis中没有，从数据库获取并初始化
	// TODO: 实现通过Consumer名称获取配额的逻辑
	return m.defaultQuota, nil
}

// ConsumeQuota 消费配额
func (m *Manager) ConsumeQuota(c *gin.Context, consumerName string, tokens int64) (bool, error) {
	if !m.enable {
		return true, nil
	}

	// 使用Redis Lua脚本原子扣减
	redisKey := common.BuildQuotaKey(consumerName)
	result, err := m.redisClient.Eval(ratelimit.ConsumeQuotaScript, 1, []string{redisKey}, []interface{}{tokens})
	if err != nil {
		return false, fmt.Errorf("failed to consume quota: %w", err)
	}

	// 检查返回值（1=成功，0=配额不足）
	if intResult, ok := result.(int64); ok {
		return intResult == 1, nil
	}

	return false, errors.New("invalid quota response")
}

// RefreshQuota 刷新配额
func (m *Manager) RefreshQuota(c *gin.Context, consumerName string, quota int64) error {
	if !m.enable {
		return nil
	}

	redisKey := common.BuildQuotaKey(consumerName)
	if err := m.redisClient.Set(redisKey, quota, int(m.quotaTTL)); err != nil {
		return fmt.Errorf("failed to refresh quota: %w", err)
	}

	return nil
}

// DeltaQuota 增减配额
func (m *Manager) DeltaQuota(c *gin.Context, consumerName string, delta int64) error {
	if !m.enable {
		return nil
	}

	redisKey := common.BuildQuotaKey(consumerName)

	// 先获取当前配额
	quotaStr, err := m.redisClient.Get(redisKey)
	if err != nil {
		return fmt.Errorf("failed to get quota: %w", err)
	}

	currentQuota := parseInt64(quotaStr)
	newQuota := currentQuota + delta

	// 设置新配额
	if err := m.redisClient.Set(redisKey, newQuota, int(m.quotaTTL)); err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	return nil
}

// InitializeQuota 初始化Consumer配额
func (m *Manager) InitializeQuota(c *gin.Context, consumerID int64) error {
	// 注意：dao.AIQuota需要通过外部传递或定义独立结构
	// 这里简化处理，只初始化Redis
	consumerName := fmt.Sprintf("consumer_%d", consumerID)
	if err := m.RefreshQuota(c, consumerName, m.defaultQuota); err != nil {
		return fmt.Errorf("failed to initialize quota in redis: %w", err)
	}

	return nil
}

// CheckAndResetQuota 检查并重置配额（周期性重置）
func (m *Manager) CheckAndResetQuota(c *gin.Context) error {
	// TODO: 实现周期性配额重置逻辑
	// 根据reset_cycle和last_reset_time判断是否需要重置
	// 如果需要重置，则更新quota_used=0和quota_remaining=quota_total
	return nil
}

// IsEnabled 检查配额是否启用
func (m *Manager) IsEnabled() bool {
	return m.enable
}

// GetDefaultQuota 获取默认配额
func (m *Manager) GetDefaultQuota() int64 {
	return m.defaultQuota
}

// GetQuotaTTL 获取配额TTL
func (m *Manager) GetQuotaTTL() int64 {
	return m.quotaTTL
}

// parseInt64 字符串转int64
func parseInt64(s string) int64 {
	var result int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		}
	}
	return result
}

// 错误定义
var (
	ErrQuotaExceeded   = errors.New("quota exceeded")
	ErrQuotaNotFound    = errors.New("quota not found")
	ErrConsumerNotFound = errors.New("consumer not found")
	ErrServiceNotFound  = errors.New("service not found")
)
