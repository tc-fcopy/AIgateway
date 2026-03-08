package dto

// AIConsumerInput Consumer输入参数
type AIConsumerInput struct {
	ID          int64  `json:"id" form:"id"`
	Name        string  `json:"consumer_name" form:"consumer_name" binding:"required"`
	Credential  string  `json:"credential" form:"credential" binding:"required"`
	Type        string  `json:"consumer_type" form:"consumer_type" binding:"required,oneof=key jwt"`
	Status      int     `json:"status" form:"status" binding:"required,oneof=0 1"`
}

// AIConsumerOutput Consumer输出参数
type AIConsumerOutput struct {
	ID          int64  `json:"id"`
	Name        string  `json:"consumer_name"`
	Credential  string  `json:"credential"`
	Type        string  `json:"consumer_type"`
	Status      int     `json:"status"`
	StatusText  string  `json:"status_text"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// GetStatusText 获取状态文本
func GetStatusText(status int) string {
	switch status {
	case 0:
		return "禁用"
	case 1:
		return "启用"
	default:
		return "未知"
	}
}

// AIConsumerListInput Consumer列表查询参数
type AIConsumerListInput struct {
	PageNum   int    `json:"page_num" form:"page_num"`
	PageSize  int    `json:"page_size" form:"page_size"`
	Name      string  `json:"consumer_name" form:"consumer_name"`
	Type      string  `json:"consumer_type" form:"consumer_type"`
	Status     *int   `json:"status" form:"status"`
}

// AIConsumerListOutput Consumer列表输出参数
type AIConsumerListOutput struct {
	List      []AIConsumerOutput `json:"list"`
	Total     int64              `json:"total"`
	PageNum   int                 `json:"page_num"`
	PageSize  int                 `json:"page_size"`
}
