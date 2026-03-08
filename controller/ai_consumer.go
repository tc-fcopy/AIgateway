package controller

import (
	"strconv"
	"time"

	"gateway/ai_gateway/consumer"
	"gateway/dao"
	"gateway/dto"
	"gateway/golang_common/lib"
	"gateway/middleware"
	"github.com/gin-gonic/gin"
)

func AIConsumerRegister(group *gin.RouterGroup) {
	group.GET("/consumer/list", AIConsumerPageList)
	group.POST("/consumer/add", AIConsumerAdd)
	group.PUT("/consumer/update/:id", AIConsumerUpdate)
	group.DELETE("/consumer/delete/:id", AIConsumerDelete)
	group.POST("/consumer/reload", AIConsumerReload)
}

func AIConsumerPageList(c *gin.Context) {
	var params dto.AIConsumerListInput
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

	q := tx.WithContext(c).Model(&dao.AIConsumer{})
	if params.Name != "" {
		q = q.Where("consumer_name LIKE ?", "%"+params.Name+"%")
	}
	if params.Type != "" {
		q = q.Where("consumer_type = ?", params.Type)
	}
	if params.Status != nil {
		q = q.Where("status = ?", *params.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	var rows []dao.AIConsumer
	if err := q.Order("id DESC").
		Limit(params.PageSize).
		Offset((params.PageNum - 1) * params.PageSize).
		Find(&rows).Error; err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	out := make([]dto.AIConsumerOutput, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.AIConsumerOutput{
			ID:         row.ID,
			Name:       row.Name,
			Credential: row.Credential,
			Type:       row.Type,
			Status:     row.Status,
			StatusText: dto.GetStatusText(row.Status),
			CreatedAt:  row.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  row.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	middleware.ResponseSuccess(c, dto.AIConsumerListOutput{
		List:     out,
		Total:    total,
		PageNum:  params.PageNum,
		PageSize: params.PageSize,
	})
}

func AIConsumerAdd(c *gin.Context) {
	var params dto.AIConsumerInput
	if err := c.ShouldBindJSON(&params); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}

	now := time.Now()
	row := &dao.AIConsumer{
		Name:       params.Name,
		Credential: params.Credential,
		Type:       params.Type,
		Status:     params.Status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	if err := tx.WithContext(c).Create(row).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	if err := reloadConsumerCache(); err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	middleware.ResponseSuccess(c, dto.AIConsumerOutput{
		ID:         row.ID,
		Name:       row.Name,
		Credential: row.Credential,
		Type:       row.Type,
		Status:     row.Status,
		StatusText: dto.GetStatusText(row.Status),
		CreatedAt:  row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  row.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

func AIConsumerUpdate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		middleware.ResponseError(c, 4000, err)
		return
	}

	var params dto.AIConsumerInput
	if err := c.ShouldBindJSON(&params); err != nil {
		middleware.ResponseError(c, 4001, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	row := &dao.AIConsumer{}
	if err := tx.WithContext(c).Where("id = ?", id).First(row).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	row.Name = params.Name
	row.Credential = params.Credential
	row.Type = params.Type
	row.Status = params.Status
	row.UpdatedAt = time.Now()

	if err := tx.WithContext(c).Save(row).Error; err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	if err := reloadConsumerCache(); err != nil {
		middleware.ResponseError(c, 5004, err)
		return
	}

	middleware.ResponseSuccess(c, dto.AIConsumerOutput{
		ID:         row.ID,
		Name:       row.Name,
		Credential: row.Credential,
		Type:       row.Type,
		Status:     row.Status,
		StatusText: dto.GetStatusText(row.Status),
		CreatedAt:  row.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  row.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

func AIConsumerDelete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		middleware.ResponseError(c, 4000, err)
		return
	}

	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 5001, err)
		return
	}

	if err := tx.WithContext(c).Delete(&dao.AIConsumer{}, id).Error; err != nil {
		middleware.ResponseError(c, 5002, err)
		return
	}

	if err := reloadConsumerCache(); err != nil {
		middleware.ResponseError(c, 5003, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{"message": "delete success"})
}

func AIConsumerReload(c *gin.Context) {
	if err := reloadConsumerCache(); err != nil {
		middleware.ResponseError(c, 5000, err)
		return
	}
	middleware.ResponseSuccess(c, gin.H{"message": "reload success"})
}

func reloadConsumerCache() error {
	rows, err := dao.LoadAIConsumers()
	if err != nil {
		return err
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
