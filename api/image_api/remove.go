package image_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"os"
	"path"

	"github.com/gin-gonic/gin"
)

func (ImageApi) ImageRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var image models.ImageModel
	if err := global.DB.Take(&image, idCr.ID).Error; err != nil {
		res.FailWithMsg("图片不存在", c)
		return
	}

	if err := global.DB.Delete(&image).Error; err != nil {
		res.FailWithMsg("删除图片失败", c)
		return
	}

	savePath := path.Join("uploads", image.Path)
	_ = os.Remove(savePath)

	res.OKWithMsg("删除成功", c)
}
