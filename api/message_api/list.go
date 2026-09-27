package message_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type MessageListRequest struct {
	common.PageInfo
	IsRead *bool `form:"isRead"` // 不传：全部；true/false：按已读筛选
}

func (MessageApi) MessageListView(c *gin.Context) {
	var cr MessageListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	query := global.DB.Where("rev_user_id = ?", claims.UserID)
	if cr.IsRead != nil {
		query = query.Where("is_read = ?", *cr.IsRead)
	}

	list, count, err := common.ListQuery(models.UserMessageModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        query,
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg(err.Error(), c)
		return
	}
	res.OKWithList(list, count, c)
}
