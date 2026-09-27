package message_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (MessageApi) MessageUnreadCountView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var count int64
	err := global.DB.Model(&models.UserMessageModel{}).
		Where("rev_user_id = ? AND is_read = ?", claims.UserID, false).
		Count(&count).Error
	if err != nil {
		res.FailWithMsg("获取未读数失败", c)
		return
	}
	res.OKWithData(gin.H{"count": count}, c)
}
