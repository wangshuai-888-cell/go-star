package article_api

import (
	"context"
	"strings"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/ai_service"
	"go-star/service/bloom_service"
	"go-star/service/redis_service/redis_tag"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm/clause"
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
	// 已发布的文章，异步做 AI 增强（不阻塞接口返回）
	if status == int8(enum.ArticleStatusPublished) {
		asyncEnhanceArticle(article.ID, article.Title, article.Content, article.Abstract, article.TagList)
	}
	res.OKWithMsg("创建成功", c)
}

// enhanceArticle 文章发布后的 AI 增强，三步彼此独立
// 任何一步失败都只是少一样东西，不会连带跳过后面
// 必须在 goroutine 里调用，ctx 由调用方用 context.Background() 派生
func enhanceArticle(ctx context.Context, id uint, title, content, abstract string, tagList []string) {
	// ① 摘要：用户没填才生成
	abstract = enrichSummary(ctx, id, title, content, abstract)
	// ② 标签：用户没填才生成
	enrichTags(ctx, id, title, content, tagList)
	// ③ 向量：每篇必算，搜索依赖它
	enrichEmbedding(ctx, id, title, abstract, content)
}

// asyncEnhanceArticle 起后台 goroutine 做完整 AI 增强（摘要 + 标签 + 向量）
// 两个发布入口共用：创建即发布、审核通过
// 必须在里面切 Background：请求已经返回，c.Request.Context() 已被取消
func asyncEnhanceArticle(id uint, title, content, abstract string, tagList []string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("AI 增强 goroutine panic articleID=%d: %v", id, r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		enhanceArticle(ctx, id, title, content, abstract, tagList)
	}()
}

// asyncRebuildEmbedding 内容变更后只重算向量
// 这里刻意不调 enhanceArticle：用户编辑时清空摘要/标签是明确意图
// 再跑一遍 AI 会把用户刚删掉的又填回去
func asyncRebuildEmbedding(id uint, title, abstract, content string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("向量重建 goroutine panic articleID=%d: %v", id, r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		enrichEmbedding(ctx, id, title, abstract, content)
	}()
}

// enrichSummary 用户没填摘要时用 AI 生成并写回
// 返回实际生效的摘要（可能是新生成的，也可能还是空），供后面的向量化使用
func enrichSummary(ctx context.Context, id uint, title, content, abstract string) string {
	if abstract != "" {
		return abstract
	}

	summary, err := ai_service.GenerateSummary(ctx, title, content)
	if err != nil {
		logrus.Errorf("AI 摘要生成失败 articleID=%d: %v", id, err)
		return abstract
	}
	if summary == "" {
		return abstract
	}

	// 条件更新：摘要仍为空才写，防止覆盖用户并发编辑时填的内容
	result := global.DB.Model(&models.ArticleModel{}).
		Where("id = ? AND (abstract = '' OR abstract IS NULL)", id).
		Update("abstract", summary)
	if result.Error != nil {
		logrus.Errorf("AI 摘要写入失败 articleID=%d: %v", id, result.Error)
		return abstract
	}
	if result.RowsAffected == 0 {
		logrus.Infof("AI 摘要跳过：用户已填写摘要 articleID=%d", id)
		return abstract
	}

	logrus.Infof("AI 摘要生成成功 articleID=%d 长度=%d", id, len([]rune(summary)))
	return summary
}

// enrichTags 用户没填标签时，用 AI 从已有标签库里挑几个写回
func enrichTags(ctx context.Context, id uint, title, content string, tagList []string) {
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
}

// enrichEmbedding 为文章生成语义搜索用的向量
// 每篇已发布文章都必须有，和用户是否填了摘要、标签无关
func enrichEmbedding(ctx context.Context, id uint, title, abstract, content string) {
	embedText := buildEmbedSource(title, abstract, content)

	vectors, err := ai_service.Embed(ctx, []string{embedText})
	if err != nil {
		logrus.Errorf("AI 向量生成失败 articleID=%d: %v", id, err)
		return
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		logrus.Errorf("AI 向量为空 articleID=%d", id)
		return
	}

	row := models.ArticleEmbeddingModel{
		ArticleID:  id,
		ModelName:  global.Config.AI.EmbeddingModel,
		Dim:        len(vectors[0]),
		SourceText: embedText,
		Vector:     vectors[0],
	}
	// upsert：一篇一向量，重复生成时覆盖旧值
	if err := global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"model_name", "dim", "source_text", "vector", "updated_at"}),
	}).Create(&row).Error; err != nil {
		logrus.Errorf("AI 向量写入失败 articleID=%d: %v", id, err)
		return
	}

	logrus.Infof("AI 向量生成成功 articleID=%d 维度=%d 长度=%d", id, row.Dim, len([]rune(embedText)))
}

// buildEmbedSource 拼向量化的原文
// 只用标题 + 摘要 + 正文前 300 字：全文太长没必要，主题信息集中在前部
func buildEmbedSource(title, abstract, content string) string {
	var b strings.Builder
	b.WriteString("标题：")
	b.WriteString(title)
	b.WriteString("\n")
	if abstract != "" {
		b.WriteString("摘要：")
		b.WriteString(abstract)
		b.WriteString("\n")
	}
	r := []rune(content)
	if len(r) > 300 {
		r = r[:300]
	}
	b.WriteString("正文：")
	b.WriteString(string(r))
	return b.String()
}
