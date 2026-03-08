package controller

import (
	"errors"

	"gateway/ai_gateway"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

type quotaRefreshInput struct {
	ConsumerName string `json:"consumer_name" binding:"required"`
	Quota        int64  `json:"quota" binding:"required"`
}

type quotaDeltaInput struct {
	ConsumerName string `json:"consumer_name" binding:"required"`
	Delta        int64  `json:"delta" binding:"required"`
}

func AIQuotaRegister(group *gin.RouterGroup) {
	group.GET("/quota", AIQuotaGet)
	group.POST("/quota/refresh", AIQuotaRefresh)
	group.POST("/quota/delta", AIQuotaDelta)
}

func AIQuotaGet(c *gin.Context) {
	manager := ai_gateway.GetQuotaManager()
	if manager == nil {
		middleware.ResponseError(c, 5001, errors.New("quota manager not initialized"))
		return
	}

	consumerName := c.Query("consumer_name")
	if consumerName == "" {
		middleware.ResponseError(c, 4001, errors.New("consumer_name is required"))
		return
	}

	left, err := manager.GetQuota(c, consumerName)
	if err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{
		"consumer_name": consumerName,
		"quota_left":    left,
	})
}

func AIQuotaRefresh(c *gin.Context) {
	manager := ai_gateway.GetQuotaManager()
	if manager == nil {
		middleware.ResponseError(c, 5001, errors.New("quota manager not initialized"))
		return
	}

	var input quotaRefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}

	if err := manager.RefreshQuota(c, input.ConsumerName, input.Quota); err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{"message": "ok"})
}

func AIQuotaDelta(c *gin.Context) {
	manager := ai_gateway.GetQuotaManager()
	if manager == nil {
		middleware.ResponseError(c, 5001, errors.New("quota manager not initialized"))
		return
	}

	var input quotaDeltaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}

	if err := manager.DeltaQuota(c, input.ConsumerName, input.Delta); err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{"message": "ok"})
}
