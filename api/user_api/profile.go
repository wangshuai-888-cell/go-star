package user_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"
	"go-star/utils/pwd"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	// 危险操作前先拿到并校验 token
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		res.FailWithMsg("token不能为空", c)
		return
	}
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

	// 用事务包住“更新密码 + 作废所有会话”：要么都成功，要么都回滚
	err = global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", hashPwd).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.UserSessionModel{}).
			Where("user_id = ? AND is_revoked = ?", user.ID, false).
			Update("is_revoked", true).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("修改密码失败 user=%d：%v", user.ID, err)
		res.FailWithMsg("修改密码失败", c)
		return
	}

	// DB 提交成功后再写Redis：让所有设备上的旧access立即失效
	if err := redis_jwt.SetPwdChangedAt(user.ID, time.Now()); err != nil {
		// 密码已改成功，这里失败只会退化成"等 access 自然过期"，必须如实告知
		logrus.Errorf("记录改密时间失败 user=%d：%v", user.ID, err)
		res.FailWithMsg("密码已修改，请用新密码重新登录", c)
		return
	}

	// 当前token拉黑
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

type UpdateConfRequest struct {
	OpenCollect *bool     `json:"openCollect"`
	OpenFollow  *bool     `json:"openFollow"`
	OpenFans    *bool     `json:"openFans"`
	LikeTags    *[]string `json:"likeTags"` // 兴趣标签，如 ["go","后端"]
}

func (UserApi) UpdateConfView(c *gin.Context) {
	var cr UpdateConfRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	if cr.OpenCollect == nil && cr.OpenFollow == nil && cr.OpenFans == nil && cr.LikeTags == nil {
		res.FailWithMsg("请至少传入一个配置项", c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var conf models.UserConfModel
	err := global.DB.Take(&conf, "user_id = ?", claims.UserID).Error
	if err != nil {
		conf = models.UserConfModel{
			UserID: claims.UserID,
		}
		if cr.OpenCollect != nil {
			conf.OpenCollect = *cr.OpenCollect
		}
		if cr.OpenFollow != nil {
			conf.OpenFollow = *cr.OpenFollow
		}
		if cr.OpenFans != nil {
			conf.OpenFans = *cr.OpenFans
		}
		if cr.LikeTags != nil {
			conf.LikeTags = *cr.LikeTags
		}
		if err := global.DB.Create(&conf).Error; err != nil {
			res.FailWithMsg("更新配置失败", c)
			return
		}

		res.OKWithData(conf, c)
		return
	}
	updates := map[string]any{}
	if cr.OpenCollect != nil {
		updates["open_collect"] = *cr.OpenCollect
		conf.OpenCollect = *cr.OpenCollect
	}
	if cr.OpenFollow != nil {
		updates["open_follow"] = *cr.OpenFollow
		conf.OpenFollow = *cr.OpenFollow
	}
	if cr.OpenFans != nil {
		updates["open_fans"] = *cr.OpenFans
		conf.OpenFans = *cr.OpenFans
	}
	if cr.LikeTags != nil {
		updates["like_tags"] = *cr.LikeTags
		conf.LikeTags = *cr.LikeTags
	}
	if err := global.DB.Model(&models.UserConfModel{}).
		Where("user_id = ?", claims.UserID).
		Updates(updates).Error; err != nil {
		res.FailWithMsg("更新配置失败", c)
		return
	}
	res.OKWithData(conf, c)
}
