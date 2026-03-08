package dto

import (
	"gateway/public"
	"github.com/gin-gonic/gin"
	"time"
)

type AdminInfoOutput struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	LoginTime    time.Time `json:"login_time"`
	Avatar       string    `json:"avatar"`
	Introduction string    `json:"introduction"`
	Roles        []string  `json:"roles"`
}

type ChangePwdInput struct {
	Password string `json:"password" form:"password" comment:"密码" validate:"required"`
}

func (param *ChangePwdInput) BindValidParam(c *gin.Context) (err error) {
	return public.DefaultGetValidParams(c, param)
}
