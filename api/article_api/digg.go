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

func (ArticleApi) ArticleDiggView(c *gin.Context) {
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

	var digged bool
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var row models.ArticleDiggModel
		// 获取当前用户是否给这篇文章点在过，能获取到记录就是点赞过
		err := tx.Where("user_id = ? AND article_id = ?", claims.UserID, article.ID).Take(&row).Error
		// 如果点赞过
		if err == nil {
			// 取消点赞，就是删除这条点赞记录，ArticleDiggModel存的就是点赞的数据
			if err := tx.Where("user_id = ? AND article_id = ?", claims.UserID, article.ID).
				Delete(&models.ArticleDiggModel{}).Error; err != nil {
				return err
			}
			// 点赞取消后，也要把文章的点赞数减1
			return tx.Model(&article).Update("digg_count", gorm.Expr("GREATEST(digg_count - ?, 0)", 1)).Error
		}
		if err := tx.Create(&models.ArticleDiggModel{
			UserID:    claims.UserID,
			ArticleID: article.ID,
		}).Error; err != nil {
			return err
		}
		digged = true
		// 如果没点赞过，就是新增一条点赞记录，也要把文章的点赞数加1
		return tx.Model(&article).Update("digg_count", gorm.Expr("digg_count + ?", 1)).Error
	})
	if err != nil {
		res.FailWithMsg("操作失败", c)
		return
	}
	if digged {
		redis_article.AddHotScore(article.ID, redis_article.ScoreDigg)
	} else {
		redis_article.AddHotScore(article.ID, -redis_article.ScoreDigg)
	}
	redis_article.ClearDetail(article.ID)
	_ = global.DB.Take(&article, article.ID)
	res.OKWithData(gin.H{
		"digg":      digged,
		"diggCount": article.DiggCount,
	}, c)
}
