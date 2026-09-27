package message_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

// 单条标记已读
func (MessageApi) MessageReadView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var msg models.UserMessageModel
	err := global.DB.Where("id = ? AND rev_user_id = ?", idCr.ID, claims.UserID).
		Take(&msg).Error
	if err != nil {
		res.FailWithMsg("消息不存在", c)
		return
	}

	if err := global.DB.Model(&msg).Update("is_read", true).Error; err != nil {
		res.FailWithMsg("操作失败", c)
		return
	}
	res.OKWithMsg("已读", c)
}

// 全部标记已读
func (MessageApi) MessageReadAllView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	err := global.DB.Model(&models.UserMessageModel{}).
		Where("rev_user_id = ? AND is_read = ?", claims.UserID, false).
		Update("is_read", true).Error
	if err != nil {
		res.FailWithMsg("操作失败", c)
		return
	}
	res.OKWithMsg("全部已读", c)
}
