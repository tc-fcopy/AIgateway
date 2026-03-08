package consumer

import "time"

// Consumer AI消费者模型
type Consumer struct {
	ID         int64     `json:"id" gorm:"primary_key;column:id"`
	Name       string    `json:"name" gorm:"unique_index;column:consumer_name;description:Consumer名称（唯一）"`
	Credential string    `json:"credential" gorm:"index;column:credential;description:访问凭证（API Key / JWT 标识）"`
	Type       string    `json:"type" gorm:"column:consumer_type;description:类型：key / jwt"`
	Status     int       `json:"status" gorm:"column:status;description:状态：0-禁用 1-启用"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:create_at;description:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:update_at;description:更新时间"`
}

// TableName 返回表名
func (c *Consumer) TableName() string {
	return "ai_consumer"
}

// IsEnabled 检查Consumer是否启用
func (c *Consumer) IsEnabled() bool {
	return c.Status == 1
}

// IsKeyType 检查是否为Key类型
func (c *Consumer) IsKeyType() bool {
	return c.Type == "key"
}

// IsJWTType 检查是否为JWT类型
func (c *Consumer) IsJWTType() bool {
	return c.Type == "jwt"
}
