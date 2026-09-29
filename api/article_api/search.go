package article_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"

	"github.com/gin-gonic/gin"
)

type ArticleSearchRequest struct {
	common.PageInfo
}

func (ArticleApi) ArticleSearchView(c *gin.Context) {
	var cr ArticleSearchRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	if cr.Key == "" {
		res.FailWithMsg("搜索关键词不能为空", c)
		return
	}

	query := global.DB.Where("status = ?", enum.ArticleStatusPublished)

	list, count, err := common.ListQuery(models.ArticleModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"title", "abstract", "content"},
		Where:        query,
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("搜索文章失败", c)
		return
	}
	for i := range list {
		list[i].LookCount += redis_article.UnflushedLook(list[i].ID)
	}
	res.OKWithList(list, count, c)
}
