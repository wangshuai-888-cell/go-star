package history_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type HistoryListRequest struct {
	common.PageInfo
}

func (HistoryApi) HistoryListView(c *gin.Context) {
	var cr HistoryListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	list, count, err := common.ListQuery(models.UserArticleLookHistoryModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        global.DB.Where("user_id = ?", claims.UserID),
		Preloads:     []string{"ArticleModel"},
		DefaultOrder: "updated_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取浏览历史失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
