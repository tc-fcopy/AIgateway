package controller

import (
	"errors"
	"strconv"
	"time"

	"gateway/dao"
	"gateway/dto"
	"gateway/golang_common/lib"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type aiServiceConfigRow struct {
	dao.AIServiceConfig
	ServiceName string `json:"service_name" gorm:"column:service_name"`
}

type aiServiceConfigReloadInput struct {
	ServiceID int64 `json:"service_id"`
}

func AIServiceConfigRegister(group *gin.RouterGroup) {
	group.GET("/service-config/list", AIServiceConfigList)
	group.GET("/service-config/detail", AIServiceConfigDetail)
	group.POST("/service-config/upsert", AIServiceConfigUpsert)
	group.DELETE("/service-config/delete/:service_id", AIServiceConfigDelete)
	group.POST("/service-config/reload", AIServiceConfigReload)
}

func AIServiceConfigList(c *gin.Context) {
	var params dto.AIServiceConfigListInput
	if err := c.ShouldBind(&params); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}
	if params.PageNum <= 0 {
		params.PageNum = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	q := tx.WithContext(c).
		Table("ai_service_config AS c").
		Select("c.*, s.service_name").
		Joins("LEFT JOIN gateway_service_info AS s ON s.id = c.service_id")
	if params.ServiceID > 0 {
		q = q.Where("c.service_id = ?", params.ServiceID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	var rows []aiServiceConfigRow
	if err := q.Order("c.id DESC").
		Limit(params.PageSize).
		Offset((params.PageNum - 1) * params.PageSize).
		Find(&rows).Error; err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	out := make([]dto.AIServiceConfigOutput, 0, len(rows))
	for _, row := range rows {
		out = append(out, buildAIServiceConfigOutput(&row))
	}

	middleware.ResponseSuccess(c, dto.AIServiceConfigListOutput{
		List:     out,
		Total:    total,
		PageNum:  params.PageNum,
		PageSize: params.PageSize,
	})
}

func AIServiceConfigDetail(c *gin.Context) {
	serviceID, err := strconv.ParseInt(c.Query("service_id"), 10, 64)
	if err != nil || serviceID <= 0 {
		middleware.ResponseError(c, 4000, errors.New("invalid service_id"))
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	row, err := loadAIServiceConfigByServiceID(c, tx, serviceID)
	if err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	middleware.ResponseSuccess(c, buildAIServiceConfigOutput(row))
}

func AIServiceConfigUpsert(c *gin.Context) {
	var params dto.AIServiceConfigInput
	if err := c.ShouldBindJSON(&params); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	if err := ensureServiceExists(c, tx, params.ServiceID); err != nil {
		middleware.ResponseError(c, 4002, err)
		return
	}

	model := &dao.AIServiceConfig{}
	row, err := model.GetByServiceID(c, tx, params.ServiceID)
	if err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	now := time.Now()
	if row == nil {
		row = &dao.AIServiceConfig{ServiceID: params.ServiceID, CreatedAt: now}
	}
	row.EnableKeyAuth = params.EnableKeyAuth
	row.EnableJWTAuth = params.EnableJWTAuth
	row.EnableTokenRateLimit = params.EnableTokenRateLimit
	row.EnableQuota = params.EnableQuota
	row.EnableModelRouter = params.EnableModelRouter
	row.EnableModelMapper = params.EnableModelMapper
	row.EnableCache = params.EnableCache
	row.EnableLoadBalancer = params.EnableLoadBalancer
	row.EnableObservability = params.EnableObservability
	row.EnablePromptDecorator = params.EnablePromptDecorator
	row.EnableIPRestriction = params.EnableIPRestriction
	row.EnableCORS = params.EnableCORS
	row.UpdatedAt = now

	if err := tx.WithContext(c).Save(row).Error; err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	if err := syncAIServiceConfigRuntime(params.ServiceID); err != nil {
		middleware.ResponseError(c, 5004, err)
		return
	}

	out, err := loadAIServiceConfigByServiceID(c, tx, params.ServiceID)
	if err != nil {
		middleware.ResponseError(c, 5005, err)
		return
	}
	middleware.ResponseSuccess(c, buildAIServiceConfigOutput(out))
}

func AIServiceConfigDelete(c *gin.Context) {
	serviceID, err := strconv.ParseInt(c.Param("service_id"), 10, 64)
	if err != nil || serviceID <= 0 {
		middleware.ResponseError(c, 4000, errors.New("invalid service_id"))
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	if err := tx.WithContext(c).Where("service_id = ?", serviceID).Delete(&dao.AIServiceConfig{}).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	if err := syncAIServiceConfigRuntime(serviceID); err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{"message": "delete success"})
}

func AIServiceConfigReload(c *gin.Context) {
	var input aiServiceConfigReloadInput
	_ = c.ShouldBindJSON(&input)

	if err := syncAIServiceConfigRuntime(input.ServiceID); err != nil {
		middleware.ResponseError(c, 5000, err)
		return
	}
	middleware.ResponseSuccess(c, gin.H{"message": "reload success"})
}

func ensureServiceExists(c *gin.Context, tx *gorm.DB, serviceID int64) error {
	info := &dao.ServiceInfo{}
	row, err := info.Find(c, tx, &dao.ServiceInfo{ID: serviceID, IsDelete: 0})
	if err != nil {
		return err
	}
	if row == nil || row.ID <= 0 {
		return errors.New("service not found")
	}
	return nil
}

func loadAIServiceConfigByServiceID(c *gin.Context, tx *gorm.DB, serviceID int64) (*aiServiceConfigRow, error) {
	row := &aiServiceConfigRow{}
	err := tx.WithContext(c).
		Table("ai_service_config AS c").
		Select("c.*, s.service_name").
		Joins("LEFT JOIN gateway_service_info AS s ON s.id = c.service_id").
		Where("c.service_id = ?", serviceID).
		First(row).Error
	if err != nil {
		return nil, err
	}
	return row, nil
}

func buildAIServiceConfigOutput(row *aiServiceConfigRow) dto.AIServiceConfigOutput {
	if row == nil {
		return dto.AIServiceConfigOutput{}
	}
	return dto.AIServiceConfigOutput{
		ID:                    row.ID,
		ServiceID:             row.ServiceID,
		ServiceName:           row.ServiceName,
		EnableKeyAuth:         row.EnableKeyAuth,
		EnableJWTAuth:         row.EnableJWTAuth,
		EnableTokenRateLimit:  row.EnableTokenRateLimit,
		EnableQuota:           row.EnableQuota,
		EnableModelRouter:     row.EnableModelRouter,
		EnableModelMapper:     row.EnableModelMapper,
		EnableCache:           row.EnableCache,
		EnableLoadBalancer:    row.EnableLoadBalancer,
		EnableObservability:   row.EnableObservability,
		EnablePromptDecorator: row.EnablePromptDecorator,
		EnableIPRestriction:   row.EnableIPRestriction,
		EnableCORS:            row.EnableCORS,
		CreatedAt:             row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:             row.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
