package dao

import (
	"errors"
	"time"

	"gateway/golang_common/lib"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AIConsumer is the db model for AI consumers.
type AIConsumer struct {
	ID         int64     `json:"id" gorm:"primary_key;column:id"`
	Name       string    `json:"name" gorm:"unique_index;column:consumer_name"`
	Credential string    `json:"credential" gorm:"index;column:credential"`
	Type       string    `json:"type" gorm:"column:consumer_type"`
	Status     int       `json:"status" gorm:"column:status"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:create_at"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:update_at"`
}

func (a *AIConsumer) TableName() string {
	return "ai_consumer"
}

type AIConsumerManager struct{}

var AIConsumerManagerHandler = &AIConsumerManager{}

func LoadAIConsumers() ([]AIConsumer, error) {
	c, _ := gin.CreateTestContext(nil)
	tx, err := lib.GetGormPool("default")
	if err != nil {
		return nil, err
	}

	var list []AIConsumer
	err = tx.WithContext(c).Where("status = ?", 1).Find(&list).Error
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (m *AIConsumerManager) GetByName(c *gin.Context, tx *gorm.DB, name string) (*AIConsumer, error) {
	var out AIConsumer
	err := tx.WithContext(c).Where("consumer_name = ? AND status = ?", name, 1).First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (m *AIConsumerManager) GetByCredential(c *gin.Context, tx *gorm.DB, credential string) (*AIConsumer, error) {
	var out AIConsumer
	err := tx.WithContext(c).Where("credential = ? AND status = ?", credential, 1).First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (m *AIConsumerManager) Find(c *gin.Context, tx *gorm.DB, search *AIConsumer) (*AIConsumer, error) {
	out := &AIConsumer{}
	err := tx.WithContext(c).Where(search).First(out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func (m *AIConsumerManager) Save(c *gin.Context, tx *gorm.DB, cons *AIConsumer) error {
	return tx.WithContext(c).Save(cons).Error
}

func (m *AIConsumerManager) Delete(c *gin.Context, tx *gorm.DB, id int64) error {
	return tx.WithContext(c).Delete(&AIConsumer{}, id).Error
}
