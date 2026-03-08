package dao

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AIQuota stores consumer quota stats.
type AIQuota struct {
	ID             int64     `json:"id" gorm:"primary_key;column:id"`
	ConsumerID     int64     `json:"consumer_id" gorm:"column:consumer_id"`
	QuotaTotal     int64     `json:"quota_total" gorm:"column:quota_total"`
	QuotaUsed      int64     `json:"quota_used" gorm:"column:quota_used"`
	QuotaRemaining int64     `json:"quota_remaining" gorm:"column:quota_remaining"`
	ResetCycle     string    `json:"reset_cycle" gorm:"column:reset_cycle"`
	LastResetTime  time.Time `json:"last_reset_time" gorm:"column:last_reset_time"`
	CreatedAt      time.Time `json:"created_at" gorm:"column:create_at"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"column:update_at"`
}

func (a *AIQuota) TableName() string {
	return "ai_quota"
}

func (a *AIQuota) Find(c *gin.Context, tx *gorm.DB, search *AIQuota) (*AIQuota, error) {
	out := &AIQuota{}
	err := tx.WithContext(c).Where(search).First(out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func (a *AIQuota) Save(c *gin.Context, tx *gorm.DB) error {
	return tx.WithContext(c).Save(a).Error
}

func (a *AIQuota) Update(c *gin.Context, tx *gorm.DB, deltaUsed int64) error {
	return tx.WithContext(c).Model(a).
		Where("id = ?", a.ID).
		Updates(map[string]interface{}{
			"quota_used":      gorm.Expr("quota_used + ?", deltaUsed),
			"quota_remaining": gorm.Expr("quota_remaining - ?", deltaUsed),
		}).Error
}

func (a *AIQuota) GetByConsumerID(c *gin.Context, tx *gorm.DB, consumerID int64) (*AIQuota, error) {
	out := &AIQuota{}
	err := tx.WithContext(c).Where("consumer_id = ?", consumerID).First(out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}
