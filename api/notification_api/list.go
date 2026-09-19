package notification_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type NotificationListRequest struct {
	common.PageInfo
}

func (NotificationApi) NotificationListView(c *gin.Context) {
	var cr NotificationListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	list, count, err := common.ListQuery(models.GlobalNotificationModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"title", "content"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取通知列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
