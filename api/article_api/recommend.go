package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type RecommendRequest struct {
	Limit int `form:"limit"`
}

func (ArticleApi) ArticleRecommendView(c *gin.Context) {
	var cr RecommendRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	if cr.Limit <= 0 || cr.Limit > 50 {
		cr.Limit = 10
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var conf models.UserConfModel
	_ = global.DB.Take(&conf, "user_id = ?", claims.UserID).Error // 获取用户的配置，不用处理错误，因为没有配置就展示热门文章

	// 没有兴趣标签就展示热门文章
	if len(conf.LikeTags) == 0 {
		ids, err := redis_article.TopHot(int64(cr.Limit))
		if err != nil || len(ids) == 0 {
			res.OKWithList([]models.ArticleModel{}, 0, c)
			return
		}
		var list []models.ArticleModel
		if err := global.DB.Where("id IN ? AND status = ?", ids, enum.ArticleStatusPublished).
			Find(&list).Error; err != nil {
			res.FailWithMsg("获取推荐失败", c)
			return
		}
		// 将文章ID和文章模型映射起来
		m := make(map[uint]models.ArticleModel, len(list))
		for _, a := range list {
			m[a.ID] = a
		}

		// 根据ID顺序构建推荐列表
		ordered := make([]models.ArticleModel, 0, len(ids))
		for _, id := range ids {
			if a, ok := m[id]; ok {
				a.LookCount += redis_article.UnflushedLook(a.ID)
				ordered = append(ordered, a)
			}
		}
		res.OKWithList(ordered, len(ordered), c)
		return
	}

	// 有兴趣标签就展示兴趣标签相关的文章
	query := global.DB.Model(&models.ArticleModel{}).Where("status = ?", enum.ArticleStatusPublished) // 已发布的文章
	or := global.DB.Where("1 = 0")
	for _, tag := range conf.LikeTags {
		or = or.Or("tag_list LIKE ?", "%\""+tag+"\"%")
	}
	query = query.Where(or)

	var list []models.ArticleModel
	err := query.Order("digg_count desc, created_at desc").Limit(cr.Limit).Find(&list).Error
	if err != nil {
		res.FailWithMsg("获取推荐失败", c)
		return
	}
	for i := range list {
		list[i].LookCount += redis_article.UnflushedLook(list[i].ID)
	}
	res.OKWithList(list, len(list), c)
}
