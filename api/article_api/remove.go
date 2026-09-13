package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (ArticleApi) ArticleRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	err := c.ShouldBindUri(&idCr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var article models.ArticleModel
	err = global.DB.Take(&article, idCr.ID).Error
	if err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}

	if article.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}

	err = global.DB.Delete(&article).Error
	if err != nil {
		res.FailWithMsg("删除失败", c)
		return
	}
	res.OKWithMsg("删除成功", c)
}
