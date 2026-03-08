package dto

// AIServiceConfigInput is the upsert payload for ai_service_config.
type AIServiceConfigInput struct {
	ServiceID             int64 `json:"service_id" binding:"required"`
	EnableKeyAuth         bool  `json:"enable_key_auth"`
	EnableJWTAuth         bool  `json:"enable_jwt_auth"`
	EnableTokenRateLimit  bool  `json:"enable_token_ratelimit"`
	EnableQuota           bool  `json:"enable_quota"`
	EnableModelRouter     bool  `json:"enable_model_router"`
	EnableModelMapper     bool  `json:"enable_model_mapper"`
	EnableCache           bool  `json:"enable_cache"`
	EnableLoadBalancer    bool  `json:"enable_loadbalancer"`
	EnableObservability   bool  `json:"enable_observability"`
	EnablePromptDecorator bool  `json:"enable_prompt_decorator"`
	EnableIPRestriction   bool  `json:"enable_ip_restriction"`
	EnableCORS            bool  `json:"enable_cors"`
}

// AIServiceConfigListInput is the query payload for listing ai_service_config rows.
type AIServiceConfigListInput struct {
	PageNum   int   `json:"page_num" form:"page_num"`
	PageSize  int   `json:"page_size" form:"page_size"`
	ServiceID int64 `json:"service_id" form:"service_id"`
}

// AIServiceConfigOutput is the API output shape for ai_service_config.
type AIServiceConfigOutput struct {
	ID                    int64  `json:"id"`
	ServiceID             int64  `json:"service_id"`
	ServiceName           string `json:"service_name"`
	EnableKeyAuth         bool   `json:"enable_key_auth"`
	EnableJWTAuth         bool   `json:"enable_jwt_auth"`
	EnableTokenRateLimit  bool   `json:"enable_token_ratelimit"`
	EnableQuota           bool   `json:"enable_quota"`
	EnableModelRouter     bool   `json:"enable_model_router"`
	EnableModelMapper     bool   `json:"enable_model_mapper"`
	EnableCache           bool   `json:"enable_cache"`
	EnableLoadBalancer    bool   `json:"enable_loadbalancer"`
	EnableObservability   bool   `json:"enable_observability"`
	EnablePromptDecorator bool   `json:"enable_prompt_decorator"`
	EnableIPRestriction   bool   `json:"enable_ip_restriction"`
	EnableCORS            bool   `json:"enable_cors"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

// AIServiceConfigListOutput is the paged list output for ai_service_config.
type AIServiceConfigListOutput struct {
	List     []AIServiceConfigOutput `json:"list"`
	Total    int64                   `json:"total"`
	PageNum  int                     `json:"page_num"`
	PageSize int                     `json:"page_size"`
}
