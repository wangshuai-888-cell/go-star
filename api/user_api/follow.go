package user_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (UserApi) UserFollowView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	if idCr.ID == claims.UserID {
		res.FailWithMsg("不能关注自己", c)
		return
	}

	var target models.UserModel
	if err := global.DB.Take(&target, idCr.ID).Error; err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	var followed bool
	var row models.UserFollowModel
	err := global.DB.Where("user_id = ? AND follow_user_id = ?", claims.UserID, target.ID).
		Take(&row).Error
	if err == nil {
		if err := global.DB.Where("user_id = ? AND follow_user_id = ?", claims.UserID, target.ID).
			Delete(&models.UserFollowModel{}).Error; err != nil {
			res.FailWithMsg("操作失败", c)
			return
		}
	} else {
		if err := global.DB.Create(&models.UserFollowModel{
			UserID:       claims.UserID,
			FollowUserID: target.ID,
		}).Error; err != nil {
			res.FailWithMsg("操作失败", c)
			return
		}
		followed = true
	}

	res.OKWithData(gin.H{"followed": followed}, c)
}

type FollowListRequest struct {
	common.PageInfo
}

func (UserApi) UserFollowListView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr FollowListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	if err := global.DB.Take(&user, idCr.ID).Error; err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	if user.ID != claims.UserID && claims.Role != enum.AdminRole {
		if !canViewFollow(user.ID) {
			res.FailWithMsg("该用户未公开关注列表", c)
			return
		}
	}

	list, count, err := common.ListQuery(models.UserFollowModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        global.DB.Where("user_id = ?", user.ID),
		Preloads:     []string{"FollowUserModel"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取关注列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}

func (UserApi) UserFansListView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr FollowListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	if err := global.DB.Take(&user, idCr.ID).Error; err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	if user.ID != claims.UserID && claims.Role != enum.AdminRole {
		if !canViewFans(user.ID) {
			res.FailWithMsg("该用户未公开粉丝列表", c)
			return
		}
	}

	list, count, err := common.ListQuery(models.UserFollowModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        global.DB.Where("follow_user_id = ?", user.ID),
		Preloads:     []string{"UserModel"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取粉丝列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}

func canViewFollow(userID uint) bool {
	var conf models.UserConfModel
	err := global.DB.Take(&conf, "user_id = ?", userID).Error
	return openFollowAllowed(err == nil, conf.OpenFollow)
}

func canViewFans(userID uint) bool {
	var conf models.UserConfModel
	err := global.DB.Take(&conf, "user_id = ?", userID).Error
	return openFansAllowed(err == nil, conf.OpenFans)
}

func openFollowAllowed(confExists, openFollow bool) bool {
	if !confExists {
		return false
	}
	return openFollow
}

func openFansAllowed(confExists, openFans bool) bool {
	if !confExists {
		return false
	}
	return openFans
}
