package collect_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (CollectApi) CollectRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var collect models.CollectModel
	if err := global.DB.Take(&collect, idCr.ID).Error; err != nil {
		res.FailWithMsg("收藏夹不存在", c)
		return
	}
	if collect.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}

	// 提到事务外面，成功后还要用这些文章 ID 清缓存
	var rows []models.UserArticleCollectModel
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collect_id = ?", collect.ID).Find(&rows).Error; err != nil {
			return err
		}
		// 删除收藏夹前，先把夹内文章的收藏数减 1
		for _, row := range rows {
			if err := tx.Model(&models.ArticleModel{}).
				Where("id = ?", row.ArticleID).
				Update("collect_count", gorm.Expr("GREATEST(collect_count - ?, 0)", 1)).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("collect_id = ?", collect.ID).
			Delete(&models.UserArticleCollectModel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&collect).Error
	})
	if err != nil {
		res.FailWithMsg("删除收藏夹失败", c)
		return
	}

	// 事务成功后，清掉这些文章的详情缓存
	for _, row := range rows {
		redis_article.ClearDetail(row.ArticleID)
	}
	res.OKWithMsg("删除成功", c)
}
