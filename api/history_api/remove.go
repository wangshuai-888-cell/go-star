package history_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func (HistoryApi) HistoryRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var history models.UserArticleLookHistoryModel
	if err := global.DB.Take(&history, idCr.ID).Error; err != nil {
		res.FailWithMsg("浏览历史不存在", c)
		return
	}
	if history.UserID != claims.UserID {
		res.FailWithMsg("权限不足", c)
		return
	}
	if err := global.DB.Delete(&history).Error; err != nil {
		res.FailWithMsg("删除浏览历史失败", c)
		return
	}
	res.OKWithMsg("删除浏览历史成功", c)
}

func (HistoryApi) HistoryClearView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	if err := global.DB.Where("user_id = ?", claims.UserID).Delete(&models.UserArticleLookHistoryModel{}).Error; err != nil {
		res.FailWithMsg("清空浏览历史失败", c)
		return
	}
	res.OKWithMsg("清空浏览历史成功", c)
}
