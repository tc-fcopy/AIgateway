package controller

import (
	"encoding/json"
	"errors"
	"gateway/dao"
	"gateway/dto"
	"gateway/golang_common/lib"
	"gateway/middleware"
	"gateway/public"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"time"
)

type AdminLoginController struct {
}

func AdminLoginRegister(group *gin.RouterGroup) {
	adminLoginController := &AdminLoginController{}
	group.POST("/login", adminLoginController.AdminLogin)
	group.GET("/login_out", adminLoginController.AdminLoginOut)
}

func (admlog *AdminLoginController) AdminLogin(c *gin.Context) {
	params := &dto.AdminLoginInput{}
	if err := params.BindValidParam(c); err != nil {
		middleware.ResponseError(c, 2000, err)
		return
	}
	// 1. 取得管理员信�?	// 2. admin.salt + params.password sha256 => saltPassword
	// 3. saltoassword == admininfo.password
	admin := &dao.Admin{}
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}

	admin, err = admin.LoginCheck(c, tx, params)
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}
	sessInfo := &dto.AdminSessionInfo{
		ID:        admin.ID,
		UserName:  admin.UserName,
		LoginTime: time.Now(),
	}
	// 设置session
	sessBts, err := json.Marshal(sessInfo)
	if err != nil {
		middleware.ResponseError(c, 2003, err)
		return
	}
	sess := sessions.Default(c)
	sess.Set(public.AdminSessionInfoKey, string(sessBts)) // 保存为字符串
	err = sess.Save()
	if err != nil {
		middleware.ResponseError(c, 2004, errors.New("failed to save session"))
		return
	}
	out := &dto.AdminLoginOutput{
		Token: admin.UserName}

	middleware.ResponseSuccess(c, out)
}

func (admlog *AdminLoginController) AdminLoginOut(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Delete(public.AdminSessionInfoKey)
	sess.Save()
	middleware.ResponseSuccess(c, "")
}
