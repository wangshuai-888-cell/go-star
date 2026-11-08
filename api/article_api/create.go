package article_api

import (
	"context"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/ai_service"
	"go-star/service/bloom_service"
	"go-star/service/redis_service/redis_tag"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ArticleCreateRequest struct {
	Title       string     `json:"title" binding:"required"`      // 标题
	Abstract    string     `json:"abstract"`                      // 简介
	Content     string     `json:"content" binding:"required"`    // 内容
	CategoryID  uint       `json:"categoryID" binding:"required"` // 分类ID
	TagList     []string   `json:"tagList"`                       // 标签列表
	Cover       string     `json:"cover"`                         // 封面
	OpenComment bool       `json:"openComment"`                   // 是否开启评论
	Status      int8       `json:"status"`                        // 1草稿 2审核中 3已发布
	PublishAt   *time.Time `json:"publishAt"`                     // 可选，定时发布时间
}

func (ArticleApi) ArticleCreateView(c *gin.Context) {
	var cr ArticleCreateRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	if cr.PublishAt != nil && !cr.PublishAt.After(time.Now()) {
		res.FailWithMsg("定时发布时间必须是未来时间", c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	// 分类是否存在
	var category models.CategoryModel
	err = global.DB.Take(&category, cr.CategoryID).Error // 这里默认是按照主键查
	if err != nil {
		res.FailWithMsg("分类不存在", c)
		return
	}

	status := int8(enum.ArticleStatusDraft)
	if cr.Status != 0 {
		status = cr.Status
	}

	if !enum.HasPerm(claims.Role, enum.ArticlePublishDirect) {
		if status != int8(enum.ArticleStatusDraft) {
			res.FailWithMsg("只能存为草稿", c)
			return
		}
	}

	article := models.ArticleModel{
		Title:       cr.Title,
		Abstract:    cr.Abstract,
		Content:     cr.Content,
		CategoryID:  cr.CategoryID,
		TagList:     cr.TagList,
		Cover:       cr.Cover,
		UserID:      claims.UserID,
		OpenComment: cr.OpenComment,
		Status:      status,
		PublishAt:   cr.PublishAt,
	}
	err = global.DB.Create(&article).Error
	if err != nil {
		res.FailWithMsg("创建失败", c)
		return
	}
	if status == int8(enum.ArticleStatusPublished) {
		if err := redis_tag.AddArticleTags(article.ID, cr.TagList); err != nil {
			logrus.Errorf("标签索引写入失败 articleID=%d: %v", article.ID, err)
		}
	}
	bloom_service.Broadcast(article.ID)
	if status == int8(enum.ArticleStatusPublished) && (article.Abstract == "" || len(article.TagList) == 0) {
		go func(id uint, title, content, abstract string, tagList []string) {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("AI 增强 goroutine panic articleID=%d: %v", id, r)
				}
			}()
			// 必须用 context.Background()：请求结束后 c.Request.Context() 会被取消
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// ===== ① 摘要 =====
			if abstract == "" {
				summary, err := ai_service.GenerateSummary(ctx, title, content)
				if err != nil {
					logrus.Errorf("AI 摘要生成失败 articleID=%d: %v", id, err)
				} else if summary != "" {
					// 条件更新：摘要仍为空才写，防止覆盖用户编辑时填的内容
					result := global.DB.Model(&models.ArticleModel{}).
						Where("id = ? AND (abstract = '' OR abstract IS NULL)", id).
						Update("abstract", summary)
					if result.Error != nil {
						logrus.Errorf("AI 摘要写入失败 articleID=%d: %v", id, result.Error)
					} else if result.RowsAffected == 0 {
						logrus.Infof("AI 摘要跳过：用户已填写摘要 articleID=%d", id)
					} else {
						logrus.Infof("AI 摘要生成成功 articleID=%d 长度=%d", id, len([]rune(summary)))
					}
				}
			}

			// ===== ② 标签 =====
			if len(tagList) > 0 {
				return
			}
			candidates, err := redis_tag.ListAllTags(50)
			if err != nil {
				logrus.Errorf("AI 标签候选集查询失败 articleID=%d: %v", id, err)
				return
			}
			logrus.Infof("AI 标签候选集 articleID=%d 数量=%d", id, len(candidates))

			tags, err := ai_service.GenerateTags(ctx, title, content, candidates)
			if err != nil {
				logrus.Errorf("AI 标签生成失败 articleID=%d: %v", id, err)
				return
			}
			if len(tags) == 0 {
				logrus.Infof("AI 标签为空，跳过 articleID=%d", id)
				return
			}

			// 条件更新：tag_list 仍为空才写，防止覆盖用户编辑时填的标签
			// 必须用 Updates + 结构体，TagList 有 serializer:json，用 Update("tag_list", ...) 不会走序列化
			result := global.DB.Model(&models.ArticleModel{}).
				Where("id = ? AND (tag_list = '[]' OR tag_list = '' OR tag_list IS NULL)", id).
				Select("tag_list").
				Updates(models.ArticleModel{TagList: tags})
			if result.Error != nil {
				logrus.Errorf("AI 标签写入文章失败 articleID=%d: %v", id, result.Error)
				return
			}
			if result.RowsAffected == 0 {
				logrus.Infof("AI 标签跳过：用户已填写标签 articleID=%d", id)
				return
			}

			// 写 tag 表 + article_tag 表 + Redis 双向索引
			if err := redis_tag.AddArticleTags(id, tags); err != nil {
				logrus.Errorf("AI 标签索引写入失败 articleID=%d: %v", id, err)
				return
			}
			logrus.Infof("AI 标签生成成功 articleID=%d tags=%v", id, tags)
		}(article.ID, article.Title, article.Content, article.Abstract, article.TagList)
	}
	res.OKWithMsg("创建成功", c)
}
