package article_service

import (
	"context"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"time"

	"github.com/sirupsen/logrus"
)

// BackfillEmbedding 为"已发布但没有向量"的文章批量补向量
// 供 go run . -embed 调用，一次性任务
func BackfillEmbedding(ctx context.Context) {
	var list []models.ArticleModel

	// 1. 查“已发布 且 没有向量”的文章
	err := global.DB.
		Where("status = ?", enum.ArticleStatusPublished).
		Where("id NOT IN (?)", global.DB.Model(&models.ArticleEmbeddingModel{}).Select("article_id")).
		Find(&list).Error
	// 2. 查询失败就停，不要带着空列表往下走
	if err != nil {
		logrus.Errorf("查询待回填文章失败: %v", err)
		return
	}

	// 3. 打印数量
	logrus.Infof("待回填向量 %d 篇", len(list))

	var ok int
	var failed []uint

	for i, a := range list {
		// 每篇单独超时，不能共用一个
		itemCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := enrichEmbedding(itemCtx, a.ID, a.Title, a.Abstract, a.Content)
		cancel()

		// 调 enrichEmbedding，失败就记下 id 继续下一篇
		if err != nil {
			failed = append(failed, a.ID)
			continue // 不能 return，一篇失败不该毁掉后面 25 篇
		}
		ok++

		// 打进度：第 i 篇 / 共 len(list) 篇
		logrus.Infof("回填进度 %d/%d articleID=%d", i+1, len(list), a.ID)

		// 歇一下再打下一篇
		time.Sleep(100 * time.Millisecond)
	}

	logrus.Infof("回填完成：成功 %d 篇，失败 %d 篇", ok, len(failed))
	if len(failed) > 0 {
		logrus.Warnf("失败的 articleID: %v", failed)
	}
}
