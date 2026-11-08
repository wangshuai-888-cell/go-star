package comment_api

import (
	"context"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/ai_service"
	"go-star/service/message_service"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CommentCreateRequest struct {
	Content  string `json:"content" binding:"required,max=256"`
	ParentID *uint  `json:"parentID"`
}

func (CommentApi) CommentCreateView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr CommentCreateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var article models.ArticleModel
	if err := global.DB.Take(&article, idCr.ID).Error; err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}
	if !article.OpenComment {
		res.FailWithMsg("评论已关闭", c)
		return
	}
	// AI 内容审核：同步调用，拦截违规评论（不入库）
	// 超时3秒。正常响应1-2秒，用户请求上下文，用户断开时AI调用同步取消
	auditStart := time.Now()
	auditCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	passed, auditErr := ai_service.AuditComment(auditCtx, cr.Content)
	cancel()

	// 统一记录审核结果，便于统计拦截率
	logrus.Infof("评论审核完成 articleID=%d userID=%d 结果=%v 耗时=%v",
		article.ID, claims.UserID, passed, time.Since(auditStart))

	if auditErr != nil {
		// fail-open：AI 不可用时放行，只记日志，避免误伤正常用户
		logrus.Warnf("评论审核服务不可用，降级放行 articleID=%d userID=%d: %v",
			article.ID, claims.UserID, auditErr)
	} else if !passed {
		logrus.Infof("评论审核拦截 articleID=%d userID=%d 内容前20字=%.20s",
			article.ID, claims.UserID, cr.Content)
		res.FailWithMsg("评论内容不符合社区规范，请修改后重试", c)
		return
	}
	comment := models.CommentModel{
		Content:   cr.Content,
		UserID:    claims.UserID,
		ArticleID: article.ID,
		ParentID:  cr.ParentID,
	}
	if cr.ParentID != nil {
		var parent models.CommentModel
		if err := global.DB.Take(&parent, *cr.ParentID).Error; err != nil {
			res.FailWithMsg("父评论不存在", c)
			return
		}
		if parent.ArticleID != article.ID {
			res.FailWithMsg("父评论不属于该文章", c)
			return
		}

		// 时机回复对象（用于展示【回复】）
		replyToCommentID := parent.ID
		replyToUserID := parent.UserID
		comment.ReplyToCommentID = &replyToCommentID
		comment.ReplyToUserID = &replyToUserID

		var rootID uint
		if parent.RootParentID != nil {
			rootID = *parent.RootParentID
		} else {
			rootID = parent.ID
		}
		comment.ParentID = &rootID
		comment.RootParentID = &rootID
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&article).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error
	})
	if err != nil {
		res.FailWithMsg("评论失败", c)
		return
	}
	redis_article.AddHotScore(article.ID, redis_article.ScoreComment)
	redis_article.ClearDetail(article.ID)

	if comment.ReplyToUserID != nil && *comment.ReplyToUserID != claims.UserID {
		articleID := article.ID
		fromID := claims.UserID
		message_service.Push(models.UserMessageModel{
			RevUserID:  *comment.ReplyToUserID,
			Type:       enum.MessageTypeCommentReply,
			Title:      "评论回复",
			Content:    "有人回复了你的评论",
			ArticleID:  &articleID,
			FromUserID: &fromID,
		})
	}

	res.OKWithData(comment, c)
}
