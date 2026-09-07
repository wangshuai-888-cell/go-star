package category_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type CategoryCreateRequest struct {
	Title string `json:"title" binding:"required"`
}

func (CategoryApi) CategoryCreateView(c *gin.Context) {
	var cr CategoryCreateRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	err = global.DB.Create(&models.CategoryModel{
		Title:  cr.Title,
		UserID: claims.UserID,
	}).Error
	if err != nil {
		res.FailWithMsg("创建分类失败", c)
		return
	}

	res.OKWithMsg("创建分类成功", c)
}

type CategoryListRequest struct {
	common.PageInfo
	Role enum.RoleType `form:"role"`
}

func (CategoryApi) CategoryListView(c *gin.Context) {
	var cr CategoryListRequest
	err := c.ShouldBindQuery(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	list, count, err := common.ListQuery(models.CategoryModel{}, common.Options{
		PageInfo: cr.PageInfo,
		Likes:    []string{"title"},
	})

	if err != nil {
		res.FailWithMsg("获取用户列表失败", c)
		return
	}

	res.OKWithList(list, count, c)
}

type CategoryUpdateRequest struct {
	Title string `json:"title" binding:"required"`
}

func (CategoryApi) CategoryUpdateView(c *gin.Context) {
	var idCr models.IDRequest
	err := c.ShouldBindUri(&idCr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	var cr CategoryUpdateRequest
	err = c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var category models.CategoryModel
	err = global.DB.Take(&category, idCr.ID).Error
	if err != nil {
		res.FailWithMsg("分类不存在", c)
		return
	}

	// 只能改自己创建的（管理员可按需放行）
	if category.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}

	err = global.DB.Model(&category).Update("title", cr.Title).Error
	if err != nil {
		res.FailWithMsg("修改分类失败", c)
		return
	}

	res.OKWithMsg("修改分类成功", c)
}

func (CategoryApi) CategoryRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	err := c.ShouldBindUri(&idCr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var category models.CategoryModel
	err = global.DB.Take(&category, idCr.ID).Error
	if err != nil {
		res.FailWithMsg("分类不存在", c)
		return
	}

	if category.UserID != claims.UserID && claims.Role != enum.AdminRole {
		res.FailWithMsg("权限不足", c)
		return
	}

	// 若分类下有文章则不允许删除
	var count int64
	global.DB.Model(&models.ArticleModel{}).Where("category_id = ?", category.ID).Count(&count)
	if count > 0 {
		res.FailWithMsg("分类下有文章，不允许删除", c)
		return
	}

	err = global.DB.Delete(&category).Error
	if err != nil {
		res.FailWithMsg("删除分类失败", c)
		return
	}

	res.OKWithMsg("删除分类成功", c)
}
