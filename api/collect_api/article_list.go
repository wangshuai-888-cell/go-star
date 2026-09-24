package collect_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CollectArticleListRequest struct {
	common.PageInfo
}

func (CollectApi) CollectArticleListView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr CollectArticleListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
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
		if !canViewCollect(collect.UserID) {
			res.FailWithMsg("该用户未公开收藏夹", c)
			return
		}
	}

	// 注意，这里用的是文章与收藏夹的关联表，关联表虽然存的字段很少，但是能同时获取两张表中关联的数据的全部信息
	list, count, err := common.ListQuery(models.UserArticleCollectModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        global.DB.Where("collect_id = ?", collect.ID),
		Preloads:     []string{"ArticleModel"},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取收藏文章失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
