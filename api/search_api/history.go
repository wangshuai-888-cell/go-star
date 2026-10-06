package search_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (SearchApi) SearchHistoryView(c *gin.Context) {
	var cr common.PageInfo
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	list, count, err := common.ListQuery(models.SearchHistoryModel{}, common.Options{
		PageInfo:     cr,
		Where:        global.DB.Where("user_id = ?", claims.UserID),
		DefaultOrder: "searched_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取搜索历史失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
