package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/history_service"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (ArticleApi) ArticleDetailView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var article models.ArticleModel
	// 缓存优先；未命中时GetDetail内部会合并并发查库（防击穿）
	if !redis_article.GetDetail(idCr.ID, &article) {
		res.FailWithMsg("文章不存在", c)
		return
	}

	if article.Status != int8(enum.ArticleStatusPublished) {
		if article.UserID != claims.UserID && claims.Role != enum.AdminRole {
			res.FailWithMsg("文章不存在", c)
			return
		}
	}

	// 非公开文章，直接返回
	if article.Status != int8(enum.ArticleStatusPublished) {
		res.OKWithData(article, c)
		return
	}

	// 浏览数先记 Redis，攒够再写数据库
	showAdd, flush, err := redis_article.AddLook(article.ID)
	if err != nil {
		// Redis 不可用时，退回每次直接写库，保证详情还能用
		global.DB.Model(&article).Update("look_count", gorm.Expr("look_count + ?", 1))
		article.LookCount++
	} else {
		if flush > 0 {
			global.DB.Model(&article).Update("look_count", gorm.Expr("look_count + ?", flush))
			redis_article.ClearDetail(article.ID)
		}
		article.LookCount += showAdd
		redis_article.AddHotScore(article.ID, redis_article.ScoreLook)
	}

	// 浏览历史：只入队，由history_service 批量落库，不阻塞接口
	history_service.Add(claims.UserID, article.ID)

	res.OKWithData(article, c)
}
