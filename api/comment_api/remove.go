package comment_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (CommentApi) CommentRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	var comment models.CommentModel
	if err := global.DB.Take(&comment, idCr.ID).Error; err != nil {
		res.FailWithMsg("评论不存在", c)
		return
	}
	if comment.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}
		return tx.Model(&models.ArticleModel{}).
			Where("id = ?", comment.ArticleID).
			Update("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", 1)).Error
	})
	if err != nil {
		res.FailWithMsg("删除失败", c)
		return
	}
	res.OKWithMsg("删除成功", c)
}
