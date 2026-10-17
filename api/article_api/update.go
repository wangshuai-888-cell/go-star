package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/service/redis_service/redis_tag"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
)

type ArticleUpdateRequest struct {
	Title       string     `json:"title" binding:"required"`
	Abstract    string     `json:"abstract"`
	Content     string     `json:"content" binding:"required"`
	CategoryID  uint       `json:"categoryID" binding:"required"`
	TagList     []string   `json:"tagList"`
	Cover       string     `json:"cover"`
	OpenComment bool       `json:"openComment"`
	Status      int8       `json:"status"`
	PublishAt   *time.Time `json:"publishAt"` // 可选，定时发布时间
}

func (ArticleApi) ArticleUpdateView(c *gin.Context) {
	var idCr models.IDRequest
	err := c.ShouldBindUri(&idCr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	var cr ArticleUpdateRequest
	err = c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var article models.ArticleModel
	err = global.DB.Take(&article, idCr.ID).Error
	if err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}

	if article.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}

	// 分类是否存在
	var category models.CategoryModel
	err = global.DB.Take(&category, cr.CategoryID).Error
	if err != nil {
		res.FailWithMsg("分类不存在", c)
		return
	}

	status := cr.Status

	if claims.Role != enum.AdminRole {
		switch article.Status {
		case int8(enum.ArticleStatusReview), int8(enum.ArticleStatusScheduled):
			res.FailWithMsg("审核中或定时待发布不可修改", c)
			return
		case int8(enum.ArticleStatusPublished):
			// 已发布被改内容，回到草稿，重新提交
			status = int8(enum.ArticleStatusDraft)
		case int8(enum.ArticleStatusDraft), int8(enum.ArticleStatusRejected):
			// 允许改，若前端乱传status，强制仍为草稿/保持驳回后可再提交
			if cr.Status != 0 && cr.Status != int8(enum.ArticleStatusDraft) {
				res.FailWithMsg("只能保存为草稿", c)
				return
			}
			status = int8(enum.ArticleStatusDraft)
		}
	} else {
		if status == 0 {
			status = article.Status
		}
	}

	err = global.DB.Model(&article).Select(
		"title", "abstract", "content", "category_id",
		"tag_list", "cover", "open_comment", "status",
		"publish_at",
	).Updates(models.ArticleModel{
		Title:       cr.Title,
		Abstract:    cr.Abstract,
		Content:     cr.Content,
		CategoryID:  cr.CategoryID,
		TagList:     cr.TagList,
		Cover:       cr.Cover,
		OpenComment: cr.OpenComment,
		Status:      status,
		PublishAt:   cr.PublishAt,
	}).Error

	if err != nil {
		res.FailWithMsg("修改失败", c)
		return
	}

	//  标签索引同步：只维护“已发布”文章的标签
	if article.Status == int8(enum.ArticleStatusPublished) {
		if status == int8(enum.ArticleStatusPublished) {
			// 还在发布状态：对比新旧标签，增量同步
			oldTags, _ := redis_tag.GetArticleTags(article.ID)
			addTags, removeTags := redis_tag.DiffTags(oldTags, cr.TagList)
			for _, t := range removeTags {
				_ = redis_tag.RemoveArticleTag(article.ID, t)
			}
			if len(addTags) > 0 {
				_ = redis_tag.AddArticleTags(article.ID, addTags)
			}
		} else {
			// 已发布被改回草稿：清掉标签索引
			_ = redis_tag.RemoveArticleTags(article.ID)
		}
	}

	redis_article.ClearDetail(article.ID)
	res.OKWithMsg("修改成功", c)
}
