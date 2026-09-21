package article_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/service/redis_service/redis_article"

	"github.com/gin-gonic/gin"
)

type ArticleListRequest struct {
	common.PageInfo
	CategoryID uint `form:"categoryID"`
	Status     int8 `form:"status"`
}

func (ArticleApi) ArticleListView(c *gin.Context) {
	var cr ArticleListRequest
	err := c.ShouldBindQuery(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	query := global.DB.Where("")
	if cr.CategoryID > 0 {
		query = query.Where("category_id = ?", cr.CategoryID)
	}
	if cr.Status > 0 {
		query = query.Where("status = ?", cr.Status)
	}

	list, count, err := common.ListQuery(models.ArticleModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"title", "abstract"},
		Where:        query,
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取文章列表失败", c)
		return
	}
	// 列表上的 lookCount 补上还没落库的那几次数，避免和详情对不上
	for i := range list {
		list[i].LookCount += redis_article.UnflushedLook(list[i].ID)
	}
	res.OKWithList(list, count, c)
}
