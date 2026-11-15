package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"go-star/service/redis_service/redis_tag"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

	if article.UserID != claims.UserID && !enum.HasPerm(claims.Role, enum.ArticleDeleteAny) {
		res.FailWithMsg("权限不足", c)
		return
	}

	err = global.DB.Delete(&article).Error
	if err != nil {
		res.FailWithMsg("删除失败", c)
		return
	}
	redis_article.ClearLook(article.ID)
	redis_article.ClearDetail(article.ID)
	redis_article.RemoveHot(article.ID)
	_ = redis_tag.RemoveArticleTags(article.ID) // 清理标签关联表和Redis索引

	// 向量在独立表里，文章是物理删除，不手动清理就会留下孤儿行
	// 孤儿行不会污染搜索结果（查询时会按文章状态过滤），但白占内存，白算余弦
	if cerr := global.DB.Where("article_id = ?", article.ID).
		Delete(&models.ArticleEmbeddingModel{}).Error; cerr != nil {
		logrus.Errorf("向量清理失败 articleID=%d: %v", article.ID, cerr)
	}
	res.OKWithMsg("删除成功", c)
}
