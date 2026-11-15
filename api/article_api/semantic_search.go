package article_api

import (
	"context"
	"math"
	"sort"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/ai_service"

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
// 流程：query 向量化 → 与库里所有向量算余弦 → 排序 → 取前 N 条文章
func (ArticleApi) ArticleSemanticSearchView(c *gin.Context) {
	var cr SemanticSearchRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	if cr.Limit <= 0 || cr.Limit > 20 {
		cr.Limit = 10
	}

	// ① query 也要向量化，并且必须用生成文章向量时的同一个模型
	// 不同模型的向量不在同一个坐标系里，混着算出来的分数没有意义
	// 搜索是同步接口，用户就盯着屏幕等，不能沿用 Chat 的 60 秒超时
	// 不包这一层的话，AI 卡住时最坏要等 3 分钟（60s × 3 次尝试）
	embedCtx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()

	vectors, err := ai_service.Embed(embedCtx, []string{cr.Key})
	if err != nil {
		logrus.Errorf("语义搜索向量化失败 key=%s: %v", cr.Key, err)
		res.FailWithMsg("搜索服务暂不可用", c)
		return
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		res.FailWithMsg("搜索服务暂不可用", c)
		return
	}
	query := vectors[0]

	// ② 取出全库向量
	// 只 select 需要的列：source_text 是长文本，算分用不上，查出来纯浪费带宽
	var rows []models.ArticleEmbeddingModel
	err = global.DB.Model(&models.ArticleEmbeddingModel{}).
		Select("article_id", "vector").
		Where("model_name = ?", global.Config.AI.EmbeddingModel).
		Find(&rows).Error
	if err != nil {
		res.FailWithMsg("搜索失败", c)
		return
	}

	// ③ 逐个算余弦相似度
	type scored struct {
		id    uint
		score float64
	}
	list := make([]scored, 0, len(rows))
	skipped := 0
	for _, row := range rows {
		// 维度不一致说明这条是别的维度生成的，跳过而不是让整个请求崩掉
		if len(row.Vector) != len(query) {
			skipped++
			continue
		}
		list = append(list, scored{id: row.ArticleID, score: cosine(query, row.Vector)})
	}
	if skipped > 0 {
		logrus.Warnf("语义搜索跳过 %d 条维度不匹配的向量 queryDim=%d model=%s",
			skipped, len(query), global.Config.AI.EmbeddingModel)
	}
	// ④ 分数倒序，截前 N
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	if len(list) > cr.Limit {
		list = list[:cr.Limit]
	}

	// ⑤ 只查这 N 篇文章
	// 顺序很重要：先算分再查文章。反过来的话要查全站文章，白查一堆用不上的
	ids := make([]uint, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.id)
	}
	var articles []models.ArticleModel
	if len(ids) > 0 {
		if err = global.DB.
			Select("id", "title", "abstract", "status").
			Where("id IN ? AND status = ?", ids, enum.ArticleStatusPublished).
			Find(&articles).Error; err != nil {
			res.FailWithMsg("搜索失败", c)
			return
		}
	}

	// 文章可能在生成向量后被下架或删除，用 map 过滤掉，同时保住分数顺序
	articleMap := make(map[uint]models.ArticleModel, len(articles))
	for _, a := range articles {
		articleMap[a.ID] = a
	}
	hits := make([]SemanticSearchHit, 0, len(list))
	for _, item := range list {
		a, ok := articleMap[item.id]
		if !ok {
			continue
		}
		hits = append(hits, SemanticSearchHit{
			ArticleID: a.ID,
			Title:     a.Title,
			Abstract:  a.Abstract,
			Score:     math.Round(item.score*10000) / 10000, // 保留 4 位，别返回 0.7719000000001
		})
	}

	res.OKWithData(hits, c)
}

// cosine 余弦相似度：cos = (A·B) / (|A|*|B|)，范围 -1~1，越大越相似
func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
