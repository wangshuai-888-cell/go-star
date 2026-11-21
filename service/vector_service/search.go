package vector_service

import (
	"context"
	"fmt"
	"go-star/global"
	"go-star/models"
	"go-star/service/ai_service"
	"math"
	"sort"

	"github.com/sirupsen/logrus"
)

// Hit 一条检索命中：只带ID和分数
// 正文由调用方按需再查，检索阶段不需要正文，带了纯属浪费带宽
type Hit struct {
	ArticleID uint
	Score     float64
}

// search 把 query向量化后与库中所有向量算余弦，返回分数最高的前limit条
// limit <= 0时用默认10，超过50截断，防止一次性捞全库
func Search(ctx context.Context, query string, limit int) ([]Hit, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	vectors, err := ai_service.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("query 向量化失败: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("向量化返回为空")
	}
	qv := vectors[0]

	var rows []models.ArticleEmbeddingModel
	err = global.DB.Model(&models.ArticleEmbeddingModel{}).
		Select("article_id", "vector").
		Where("model_name = ?", global.Config.AI.EmbeddingModel).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("加载向量失败：%v", err)
	}

	hits := make([]Hit, 0, len(rows))
	skipped := 0
	for _, row := range rows {
		// 维度不一致说明这条是用别的模型生成的，跳过而不是让整个请求崩掉
		if len(row.Vector) != len(qv) {
			skipped++
			continue
		}
		hits = append(hits, Hit{ArticleID: row.ArticleID, Score: Cosine(qv, row.Vector)})
	}
	if skipped > 0 {
		logrus.Warnf("向量检索跳过 %d 条维度不匹配的记录 queryDim=%d model=%s", skipped, len(qv), global.Config.AI.EmbeddingModel)
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

// Cosine 余弦相似度：cos = (A·B) / (|A|*|B|)，范围 -1~1，越大越相似
func Cosine(a, b []float32) float64 {
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
