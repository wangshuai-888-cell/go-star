package banner_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type BannerListRequest struct {
	common.PageInfo
}

func (BannerApi) BannerListView(c *gin.Context) {
	var cr BannerListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	list, count, err := common.ListQuery(models.BannerModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取轮播图失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
