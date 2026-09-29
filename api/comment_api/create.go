package comment_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
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
		models.CreateUserMessage(models.UserMessageModel{
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
