package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/service/redis_service/redis_article"

	"github.com/gin-gonic/gin"
)

type HotListRequest struct {
	Limit int64 `form:"limit"`
}

func (ArticleApi) ArticleHotListView(c *gin.Context) {
	var cr HotListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	if cr.Limit <= 0 || cr.Limit > 50 {
		cr.Limit = 10
	}

	ids, err := redis_article.TopHot(cr.Limit)
	if err != nil {
		res.FailWithMsg("获取热门失败", c)
		return
	}
	if len(ids) == 0 {
		res.OKWithList([]models.ArticleModel{}, 0, c)
		return
	}

	var list []models.ArticleModel
	if err := global.DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
		res.FailWithMsg("获取热门失败", c)
		return
	}

	// Find 不保证顺序，按排行 id 再排一次
	m := make(map[uint]models.ArticleModel, len(list))
	for _, a := range list {
		m[a.ID] = a
	}
	// 把数据库查回来的文章，按照redis热榜的顺序重新排好，并且把没写进数据库中的浏览量也加上
	ordered := make([]models.ArticleModel, 0, len(ids))
	for _, id := range ids {
		if a, ok := m[id]; ok {
			a.LookCount += redis_article.UnflushedLook(a.ID)
			ordered = append(ordered, a)
		}
	}
	res.OKWithList(ordered, len(ordered), c)
}
