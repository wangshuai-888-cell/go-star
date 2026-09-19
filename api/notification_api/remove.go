package notification_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

func (NotificationApi) NotificationRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var n models.GlobalNotificationModel
	if err := global.DB.Take(&n, idCr.ID).Error; err != nil {
		res.FailWithMsg("通知不存在", c)
		return
	}
	if err := global.DB.Delete(&n).Error; err != nil {
		res.FailWithMsg("删除通知失败", c)
		return
	}
	res.OKWithMsg("删除成功", c)
}
