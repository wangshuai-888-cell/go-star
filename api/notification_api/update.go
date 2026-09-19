package notification_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type NotificationUpdateRequest struct {
	Title   string `json:"title" binding:"required,max=32"`
	Icon    string `json:"icon" binding:"max=256"`
	Content string `json:"content" binding:"required,max=64"`
	Href    string `json:"href" binding:"max=256"`
}

func (NotificationApi) NotificationUpdateView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr NotificationUpdateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var n models.GlobalNotificationModel
	if err := global.DB.Take(&n, idCr.ID).Error; err != nil {
		res.FailWithMsg("通知不存在", c)
		return
	}

	err := global.DB.Model(&n).Updates(map[string]any{
		"title":   cr.Title,
		"icon":    cr.Icon,
		"content": cr.Content,
		"href":    cr.Href,
	}).Error
	if err != nil {
		res.FailWithMsg("更新通知失败", c)
		return
	}
	res.OKWithMsg("修改成功", c)
}
