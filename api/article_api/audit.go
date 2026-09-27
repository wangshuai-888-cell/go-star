package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
)

// 作者提交审核
func (ArticleApi) ArticleSubmitView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
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
	if article.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}
	if article.Status != int8(enum.ArticleStatusDraft) && article.Status != int8(enum.ArticleStatusRejected) {
		res.FailWithMsg("只能提交审核草稿或驳回后可再提交", c)
		return
	}

	if err := global.DB.Model(&article).Update("status", enum.ArticleStatusReview).Error; err != nil {
		res.FailWithMsg("提交审核失败", c)
		return
	}
	redis_article.ClearDetail(article.ID)
	res.OKWithMsg("提交审核成功", c)
}

type ArticleAuditRequest struct {
	Pass bool `json:"pass"` // true 通过，false 驳回
}

// 管理员审核
func (ArticleApi) ArticleAuditView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr ArticleAuditRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var article models.ArticleModel
	if err := global.DB.Take(&article, idCr.ID).Error; err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}

	if article.Status != int8(enum.ArticleStatusReview) {
		res.FailWithMsg("只能审核「审核中」的文章", c)
		return
	}

	next := enum.ArticleStatusPublished
	msg := "审核通过"
	if !cr.Pass {
		next = enum.ArticleStatusRejected
		msg = "已驳回"
	} else if article.PublishAt != nil && article.PublishAt.After(time.Now()) {
		next = enum.ArticleStatusScheduled
		msg = "审核通过，已加入定时发布"
	}
	if err := global.DB.Model(&article).Update("status", next).Error; err != nil {
		res.FailWithMsg("审核失败", c)
		return
	}
	redis_article.ClearDetail(article.ID)

	articleID := article.ID
	if cr.Pass {
		content := "你的文章《" + article.Title + "》已通过审核"
		if next == enum.ArticleStatusScheduled {
			content = "你的文章《" + article.Title + "》已通过审核，将于定时时间发布"
		}
		models.CreateUserMessage(models.UserMessageModel{
			RevUserID: article.UserID,
			Type:      enum.MessageTypeAuditPass,
			Title:     "审核通过",
			Content:   content,
			ArticleID: &articleID,
		})
	} else {
		models.CreateUserMessage(models.UserMessageModel{
			RevUserID: article.UserID,
			Type:      enum.MessageTypeAuditReject,
			Title:     "审核驳回",
			Content:   "你的文章《" + article.Title + "》未通过审核",
			ArticleID: &articleID,
		})
	}

	res.OKWithMsg(msg, c)
}
