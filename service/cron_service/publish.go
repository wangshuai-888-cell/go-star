package cron_service

import (
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"time"

	"github.com/sirupsen/logrus"
)

// 扫描到期的定时文章并发布
func PublishScheduledArticles() {
	var list []models.ArticleModel
	err := global.DB.Where(
		"status = ? AND publish_at IS NOT NULL AND publish_at <= ?",
		enum.ArticleStatusScheduled,
		time.Now(),
	).Find(&list).Error
	if err != nil {
		logrus.Println("扫描到期的定时文章失败", err)
		return
	}

	for _, article := range list {
		err := global.DB.Model(&article).Update("status", enum.ArticleStatusPublished).Error
		if err != nil {
			logrus.Errorf("定时发布失败 id=%d: %v", article.ID, err)
			continue
		}
		redis_article.ClearDetail(article.ID)

		articleID := article.ID
		models.CreateUserMessage(models.UserMessageModel{
			RevUserID: article.UserID,
			Type:      enum.MessageTypeAuditPass,
			Title:     "文章已发布",
			Content:   "你的文章《" + article.Title + "》已按时发布",
			ArticleID: &articleID,
		})
		logrus.Infof("定时发布成功 id=%d title=%s", article.ID, article.Title)
	}
}

// 启动定时任务：每分钟扫一次
func Run() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			PublishScheduledArticles()
			<-ticker.C
		}
	}()
	logrus.Info("定时发布任务已启动")
}
