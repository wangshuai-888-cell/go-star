package collect_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CollectListRequest struct {
	common.PageInfo
}

func (CollectApi) CollectListView(c *gin.Context) {
	var cr CollectListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	list, count, err := common.ListQuery(models.CollectModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Likes:        []string{"title", "abstract"},
		Where:        global.DB.Where("user_id = ?", claims.UserID),
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取收藏夹列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
