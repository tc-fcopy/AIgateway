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
)

type AdminController struct {
}

func AdminRegister(group *gin.RouterGroup) {
	adminController := &AdminController{}
	group.GET("/admin_info", adminController.AdminInfo)
	group.POST("/change_pwd", adminController.AdminChangePwd)
}

func (adm *AdminController) AdminInfo(c *gin.Context) {
	sess := sessions.Default(c)
	sessInfo := sess.Get(public.AdminSessionInfoKey)

	// 统一使用string类型存储
	if sessInfo == nil {
		middleware.ResponseError(c, 2000, errors.New("user not logged in"))
		return
	}

	adminSessionInfo := &dto.AdminSessionInfo{}
	if err := json.Unmarshal([]byte(sessInfo.(string)), adminSessionInfo); err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	out := &dto.AdminInfoOutput{
		ID:           adminSessionInfo.ID,
		Name:         adminSessionInfo.UserName,
		LoginTime:    adminSessionInfo.LoginTime,
		Avatar:       ".....",
		Introduction: "I am a super administrator",
		Roles:        []string{"admin"},
	}
	middleware.ResponseSuccess(c, out)
}

func (adm *AdminController) AdminChangePwd(c *gin.Context) {
	params := &dto.ChangePwdInput{}
	if err := params.BindValidParam(c); err != nil {
		middleware.ResponseError(c, 2003, err)
		return
	}

	// 1. 读取Session数据（统一使用string类型
	sess := sessions.Default(c)
	sessInfo := sess.Get(public.AdminSessionInfoKey)
	if sessInfo == nil {
		middleware.ResponseError(c, 2004, errors.New("session expired"))
		return
	}

	adminSessionInfo := &dto.AdminSessionInfo{}
	if err := json.Unmarshal([]byte(sessInfo.(string)), adminSessionInfo); err != nil {
		middleware.ResponseError(c, 2005, err)
		return
	}

	// 2. 查询数据库中的用户信息
	tx, err := lib.GetGormPool("default")
	if err != nil {
		middleware.ResponseError(c, 2001, err)
		return
	}
	adminInfo := &dao.Admin{}
	adminInfo, err = adminInfo.Find(c, tx, (&dao.Admin{UserName: adminSessionInfo.UserName}))
	if err != nil {
		middleware.ResponseError(c, 2002, err)
		return
	}

	// 3. 生成新密码（旧Salt + 新明文密码）
	newSaltPassword := public.GenSaltPassword(adminInfo.Salt, params.Password)

	// 4. 更新密码
	adminInfo.Password = newSaltPassword
	if err := adminInfo.Save(c, tx); err != nil {
		middleware.ResponseError(c, 2007, err)
		return
	}

	middleware.ResponseSuccess(c, gin.H{"message": "Password updated successfully"})
}
