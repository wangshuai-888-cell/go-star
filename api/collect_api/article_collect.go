package collect_api

import (
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ArticleCollectRequest struct {
	CollectID uint `json:"collectID" binding:"required"`
}

func (CollectApi) ArticleCollectView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr ArticleCollectRequest
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

	var collect models.CollectModel
	if err := global.DB.Take(&collect, cr.CollectID).Error; err != nil {
		res.FailWithMsg("收藏夹不存在", c)
		return
	}
	if collect.UserID != claims.UserID {
		res.FailWithMsg("只能收藏到自己的收藏夹", c)
		return
	}

	var collected bool
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var row models.UserArticleCollectModel
		// 在文章收藏夹的关联表中，找到对应的记录
		err := tx.Where("user_id = ? AND article_id = ? AND collect_id = ?",
			claims.UserID, article.ID, collect.ID).Take(&row).Error
		// 如果有这条记录，说明已经收藏过了
		if err == nil {
			// 已收藏 -> 取消，删除这条关联记录
			if err := tx.Where("user_id = ? AND article_id = ? AND collect_id = ?",
				claims.UserID, article.ID, collect.ID).
				Delete(&models.UserArticleCollectModel{}).Error; err != nil {
				return err
			}
			// 收藏夹中的文章数量减1
			if err := tx.Model(&collect).
				Update("article_count", gorm.Expr("GREATEST(article_count - ?, 0)", 1)).Error; err != nil {
				return err
			}
			// 文章被收藏的数量减1
			return tx.Model(&article).
				Update("collect_count", gorm.Expr("GREATEST(collect_count - ?, 0)", 1)).Error
		}

		// 如果关联表中没有记录，就说明之前没收藏过，创建一条关联记录
		if err := tx.Create(&models.UserArticleCollectModel{
			UserID:    claims.UserID,
			ArticleID: article.ID,
			CollectID: collect.ID,
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}
		// 收藏夹的收藏数量加1
		collected = true
		if err := tx.Model(&collect).
			Update("article_count", gorm.Expr("article_count + ?", 1)).Error; err != nil {
			return err
		}
		// 文章被收藏的数量加1
		return tx.Model(&article).
			Update("collect_count", gorm.Expr("collect_count + ?", 1)).Error
	})
	if err != nil {
		res.FailWithMsg("操作失败", c)
		return
	}

	_ = global.DB.Take(&article, article.ID)
	res.OKWithData(gin.H{
		"collected":    collected,
		"collectCount": article.CollectCount,
	}, c)
}
