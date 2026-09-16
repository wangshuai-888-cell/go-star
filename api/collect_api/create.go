package collect_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CollectCreateRequest struct {
	Title    string `json:"title" binding:"required,max=32"`
	Abstract string `json:"abstract" binding:"required,max=256"`
	Cover    string `json:"cover" binding:"required,max=256"` // 收藏夹封面，url地址
}

func (CollectApi) CollectCreateView(c *gin.Context) {
	var cr CollectCreateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	collect := models.CollectModel{
		Title:    cr.Title,
		Abstract: cr.Abstract,
		Cover:    cr.Cover,
		UserID:   claims.UserID,
	}
	if err := global.DB.Create(&collect).Error; err != nil {
		res.FailWithMsg("创建收藏夹失败", c)
		return
	}
	res.OKWithData(collect, c)
}
