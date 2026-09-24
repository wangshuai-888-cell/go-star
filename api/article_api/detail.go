package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
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
	// 如果redis中没有，再从数据库中找
	if !redis_article.GetDetail(idCr.ID, &article) {
		if err := global.DB.Take(&article, idCr.ID).Error; err != nil {
			res.FailWithMsg("文章不存在", c)
			return
		}
		// 如果数据库中有，就存入redis中一份
		redis_article.SetDetail(article)
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

	userID := claims.UserID
	articleID := article.ID
	// go func() { ... }() 的意思是：立刻新开一条执行路径去跑大括号里的代码，当前函数不等它结束，继续往下执行res.OKWithData()，异步执行
	go func() {
		var history models.UserArticleLookHistoryModel
		err := global.DB.Where("user_id = ? AND article_id = ?", userID, articleID).Take(&history).Error
		if err != nil {
			_ = global.DB.Create(&models.UserArticleLookHistoryModel{
				UserID:    userID,
				ArticleID: articleID,
			}).Error
			return
		}
		_ = global.DB.Model(&history).Update("updated_at", gorm.Expr("NOW()")).Error
	}()

	res.OKWithData(article, c)
}
