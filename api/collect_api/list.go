package collect_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CollectListRequest struct {
	common.PageInfo
	UserID uint `form:"userID"` // 不传或传自己：看自己的；传别人：看对方的
}

func (CollectApi) CollectListView(c *gin.Context) {
	var cr CollectListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	targetID := cr.UserID
	if targetID == 0 {
		targetID = claims.UserID
	}

	if targetID != claims.UserID && claims.Role != enum.AdminRole {
		if !canViewCollect(targetID) {
			res.FailWithMsg("该用户未公开收藏夹", c)
			return
		}
	}

	list, count, err := common.ListQuery(models.CollectModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"title", "abstract"},
		Where:        global.DB.Where("user_id = ?", targetID),
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取收藏夹列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}

func canViewCollect(userID uint) bool {
	var conf models.UserConfModel
	if err := global.DB.Take(&conf, "user_id = ?", userID).Error; err != nil {
		return false
	}
	return conf.OpenCollect
}
