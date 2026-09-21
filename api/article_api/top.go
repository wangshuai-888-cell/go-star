package article_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (ArticleApi) ArticleTopView(c *gin.Context) {
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
	if article.UserID != claims.UserID {
		res.FailWithMsg("只能置顶自己的文章", c)
		return
	}

	var topped bool
	var row models.UserTopArticleModel
	err := global.DB.Where("user_id = ? AND article_id = ?", claims.UserID, article.ID).Take(&row).Error
	if err == nil {
		if err := global.DB.Where("user_id = ? AND article_id = ?", claims.UserID, article.ID).
			Delete(&models.UserTopArticleModel{}).Error; err != nil {
			res.FailWithMsg("操作失败", c)
			return
		}
	} else {
		if err := global.DB.Create(&models.UserTopArticleModel{
			UserID:    claims.UserID,
			ArticleID: article.ID,
		}).Error; err != nil {
			res.FailWithMsg("操作失败", c)
			return
		}
		topped = true
	}

	res.OKWithData(gin.H{
		"top": topped,
	}, c)
}

type TopArticleListRequest struct {
	common.PageInfo
}

func (ArticleApi) UserTopArticleListView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr TopArticleListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var user models.UserModel
	if err := global.DB.Take(&user, idCr.ID).Error; err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	list, count, err := common.ListQuery(models.UserTopArticleModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        global.DB.Where("user_id = ?", user.ID),
		Preloads:     []string{"ArticleModel"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取置顶文章失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
