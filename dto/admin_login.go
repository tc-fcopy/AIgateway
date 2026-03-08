package dto

import (
	"gateway/public"
	"github.com/gin-gonic/gin"
	"time"
)

type AdminSessionInfo struct {
	ID        int64     `json:"id"`
	UserName  string    `json:"username"`
	LoginTime time.Time `json:"login_time"`
}

type AdminLoginInput struct {
	Username string `json:"username" form:"username" comment:"管理员用户名" validate:"required,valid_username"`
	Password string `json:"password" form:"password" comment:"密码" validate:"required"`
}

func (param *AdminLoginInput) BindValidParam(c *gin.Context) (err error) {
	return public.DefaultGetValidParams(c, param)
}

type AdminLoginOutput struct {
	Token string `json:"token" form:"token" comment:"token" validate:" "`
}
