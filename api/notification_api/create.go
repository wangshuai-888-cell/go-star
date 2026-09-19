package notification_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type NotificationCreateRequest struct {
	Title   string `json:"title" binding:"required,max=32"`
	Icon    string `json:"icon" binding:"max=256"`
	Content string `json:"content" binding:"required,max=64"`
	Href    string `json:"href" binding:"max=256"`
}

func (NotificationApi) NotificationCreateView(c *gin.Context) {
	var cr NotificationCreateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	n := models.GlobalNotificationModel{
		Title:   cr.Title,
		Icon:    cr.Icon,
		Content: cr.Content,
		Href:    cr.Href,
	}
	if err := global.DB.Create(&n).Error; err != nil {
		res.FailWithMsg("创建通知失败", c)
		return
	}
	res.OKWithData(n, c)
}
