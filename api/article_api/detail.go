package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
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
	if err := global.DB.Take(&article, idCr.ID).Error; err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}

	// 浏览数+1
	// global.DB.Model(&article).Update("look_count", article.LookCount+1) // 这种写法是先增加在写入数据库，如果数据库有并发操作，可能会导致数据不准确
	global.DB.Model(&article).Update("look_count", gorm.Expr("look_count + ?", 1)) // 这种写法是先计算在写入数据库，可以避免并发操作导致数据不准确
	article.LookCount++

	var history models.UserArticleLookHistoryModel
	err := global.DB.Where("user_id = ? AND article_id = ?", claims.UserID, article.ID).Take(&history).Error
	if err != nil {
		global.DB.Create(&models.UserArticleLookHistoryModel{
			UserID:    claims.UserID,
			ArticleID: article.ID,
		})
	} else {
		global.DB.Model(&history).Update("updated_at", gorm.Expr("NOW()"))
	}

	res.OKWithData(article, c)
}
