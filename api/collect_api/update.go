package collect_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CollectUpdateRequest struct {
	Title    string `json:"title" binding:"required,max=32"`
	Abstract string `json:"abstract" binding:"required,max=256"`
	Cover    string `json:"cover" binding:"required,max=256"`
}

func (CollectApi) CollectUpdateView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr CollectUpdateRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
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
		res.FailWithMsg("无权限操作", c)
		return
	}

	err := global.DB.Model(&collect).Updates(map[string]any{
		"title":    cr.Title,
		"abstract": cr.Abstract,
		"cover":    cr.Cover,
	}).Error
	if err != nil {
		res.FailWithMsg("更新收藏夹失败", c)
		return
	}
	res.OKWithMsg("更新收藏夹成功", c)
}
