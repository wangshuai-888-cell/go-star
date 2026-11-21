package article_api

import (
	"context"
	"math"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/vector_service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SemanticSearchRequest 语义搜索入参
// 和 /articles/search 的区别：那边是 LIKE 关键词匹配，这边是向量相似度
type SemanticSearchRequest struct {
	Key   string `form:"key" binding:"required"`
	Limit int    `form:"limit"`
}

// SemanticSearchHit 单条命中
type SemanticSearchHit struct {
	ArticleID uint    `json:"articleID"`
	Title     string  `json:"title"`
	Abstract  string  `json:"abstract"`
	Score     float64 `json:"score"`
}

// ArticleSemanticSearchView 语义搜索
// 流程：检索（service 层）→ 查文章 → 按发布状态过滤 → 返回
func (ArticleApi) ArticleSemanticSearchView(c *gin.Context) {
	var cr SemanticSearchRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	if cr.Limit <= 0 || cr.Limit > 20 {
		cr.Limit = 10
	}

	// 同步接口，用户盯着屏幕等，8 秒兜底
	// 不设这个的话，AI 卡住时最坏要等 3 分钟（60s × 3 次重试）
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	hits, err := vector_service.Search(ctx, cr.Key, cr.Limit)
	if err != nil {
		logrus.Errorf("语义搜索失败 key=%s: %v", cr.Key, err)
		res.FailWithMsg("搜索服务暂不可用", c)
		return
	}
	if len(hits) == 0 {
		res.OKWithData([]SemanticSearchHit{}, c)
		return
	}

	// 只查这 N 篇文章：先算分再查文章，顺序反了就要捞全站正文
	ids := make([]uint, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ArticleID)
	}
	var articles []models.ArticleModel
	if err = global.DB.
		Select("id", "title", "abstract", "status").
		Where("id IN ? AND status = ?", ids, enum.ArticleStatusPublished).
		Find(&articles).Error; err != nil {
		res.FailWithMsg("搜索失败", c)
		return
	}

	// 文章可能在生成向量后被下架或删除，用 map 过滤掉，同时保住分数顺序
	articleMap := make(map[uint]models.ArticleModel, len(articles))
	for _, a := range articles {
		articleMap[a.ID] = a
	}
	list := make([]SemanticSearchHit, 0, len(hits))
	for _, h := range hits {
		a, ok := articleMap[h.ArticleID]
		if !ok {
			continue
		}
		list = append(list, SemanticSearchHit{
			ArticleID: a.ID,
			Title:     a.Title,
			Abstract:  a.Abstract,
			Score:     math.Round(h.Score*10000) / 10000, // 保留 4 位
		})
	}

	res.OKWithData(list, c)
}
