package banner_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type BannerUpdateRequest struct {
	Cover string `json:"cover" binding:"required,max=256"`
	Href  string `json:"href" binding:"max=256"`
}

func (BannerApi) BannerUpdateView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr BannerUpdateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var banner models.BannerModel
	if err := global.DB.Take(&banner, idCr.ID).Error; err != nil {
		res.FailWithMsg("轮播图不存在", c)
		return
	}
	err := global.DB.Model(&banner).Updates(map[string]any{
		"cover": cr.Cover,
		"href":  cr.Href,
	}).Error
	if err != nil {
		res.FailWithMsg("修改轮播图失败", c)
		return
	}
	res.OKWithMsg("修改成功", c)
}
