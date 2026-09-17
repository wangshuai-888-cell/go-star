package image_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type ImageListRequest struct {
	common.PageInfo
}

type ImageListItem struct {
	models.ImageModel
	WebPath string `json:"webPath"`
}

func (ImageApi) ImageListView(c *gin.Context) {
	var cr ImageListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	list, count, err := common.ListQuery(models.ImageModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"filename"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取图片列表失败", c)
		return
	}

	items := make([]ImageListItem, 0, len(list))
	for _, img := range list {
		items = append(items, ImageListItem{
			ImageModel: img,
			WebPath:    img.WebPath(),
		})
	}
	res.OKWithList(items, count, c)
}
