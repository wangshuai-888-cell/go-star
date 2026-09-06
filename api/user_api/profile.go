package user_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"
	"go-star/utils/pwd"

	"github.com/gin-gonic/gin"
)

// 获取用户信息接口
func (UserApi) UserInfoView(c *gin.Context) {
	// 从中间件中获取claims
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	// 按userID查库
	var user models.UserModel
	err := global.DB.Take(&user, "id = ?", claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	res.OKWithData(user, c)
}

// 修改密码
type ChangePwdRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (UserApi) ChangePwdView(c *gin.Context) {
	var cr ChangePwdRequest
	err := c.ShouldBindJSON(&cr)

	if err != nil {
		res.FailWithError(err, c)
		return
	}

	// 从token中拿当前用户
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	err = global.DB.Take(&user, claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	// 校验旧密码
	if !pwd.CheckPwd(user.Password, cr.OldPassword) {
		res.FailWithMsg("旧密码错误", c)
		return
	}

	if cr.OldPassword == cr.NewPassword {
		res.FailWithMsg("新密码不能与旧密码相同", c)
		return
	}

	// 新密码加密
	hashPwd, err := pwd.HashPwd(cr.NewPassword)
	if err != nil {
		res.FailWithMsg("密码加密失败", c)
		return
	}

	// 更新密码
	err = global.DB.Model(&user).Update("password", hashPwd).Error
	if err != nil {
		res.FailWithMsg("更新密码失败", c)
		return
	}

	// 更新后拉黑当前token，重新登录
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		res.FailWithMsg("token不能为空", c)
		return
	}
	redis_jwt.TokenBlack(token, redis_jwt.UserBlackType)

	res.OKWithMsg("修改密码成功", c)
}

// 修改用户信息
type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Abstract string `json:"abstract"` // 简介
	Avatar   string `json:"avatar"`   // 头像
}

func (UserApi) UpdateUserView(c *gin.Context) {
	var cr UpdateUserRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	err = global.DB.Take(&user, claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	err = global.DB.Model(&user).Updates(models.UserModel{
		Nickname: cr.Nickname,
		Abstract: cr.Abstract,
		Avatar:   cr.Avatar,
	}).Error
	if err != nil {
		res.FailWithMsg("更新用户信息失败", c)
		return
	}

	res.OKWithMsg("更新用户信息成功", c)
}
