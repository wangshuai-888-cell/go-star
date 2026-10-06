package search_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type SearchHotRequest struct {
	Limit int `form:"limit"`
}

func (SearchApi) SearchHotView(c *gin.Context) {
	var cr SearchHotRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	limit := cr.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var list []models.SearchHotModel
	err := global.DB.
		Order("search_count desc, last_search_at desc").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		res.FailWithMsg("获取热搜失败", c)
		return
	}
	res.OKWithData(list, c)
}
