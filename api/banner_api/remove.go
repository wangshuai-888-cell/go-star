package banner_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

func (BannerApi) BannerRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var banner models.BannerModel
	if err := global.DB.Take(&banner, idCr.ID).Error; err != nil {
		res.FailWithMsg("轮播图不存在", c)
		return
	}
	if err := global.DB.Delete(&banner).Error; err != nil {
		res.FailWithMsg("删除轮播图失败", c)
		return
	}
	res.OKWithMsg("删除成功", c)
}
