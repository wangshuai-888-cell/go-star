package banner_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type BannerCreateRequest struct {
	Cover string `json:"cover" binding:"required,max=256"`
	Href  string `json:"href" binding:"max=256"`
}

func (BannerApi) BannerCreateView(c *gin.Context) {
	var cr BannerCreateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	banner := models.BannerModel{
		Cover: cr.Cover,
		Href:  cr.Href,
	}
	if err := global.DB.Create(&banner).Error; err != nil {
		res.FailWithMsg("创建轮播图失败", c)
		return
	}

	res.OKWithData(banner, c)
}
