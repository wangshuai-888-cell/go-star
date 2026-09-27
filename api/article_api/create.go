package article_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

type ArticleCreateRequest struct {
	Title       string   `json:"title" binding:"required"`      // 标题
	Abstract    string   `json:"abstract"`                      // 简介
	Content     string   `json:"content" binding:"required"`    // 内容
	CategoryID  uint     `json:"categoryID" binding:"required"` // 分类ID
	TagList     []string `json:"tagList"`                       // 标签列表
	Cover       string   `json:"cover"`                         // 封面
	OpenComment bool     `json:"openComment"`                   // 是否开启评论
	Status      int8     `json:"status"`                        // 1草稿 2审核中 3已发布
}

func (ArticleApi) ArticleCreateView(c *gin.Context) {
	var cr ArticleCreateRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	// 分类是否存在
	var category models.CategoryModel
	err = global.DB.Take(&category, cr.CategoryID).Error // 这里默认是按照主键查
	if err != nil {
		res.FailWithMsg("分类不存在", c)
		return
	}

	status := int8(enum.ArticleStatusDraft)
	if cr.Status != 0 {
		status = cr.Status
	}

	if claims.Role != enum.AdminRole {
		if status != int8(enum.ArticleStatusDraft) {
			res.FailWithMsg("只能存为草稿", c)
			return
		}
	}

	err = global.DB.Create(&models.ArticleModel{
		Title:       cr.Title,
		Abstract:    cr.Abstract,
		Content:     cr.Content,
		CategoryID:  cr.CategoryID,
		TagList:     cr.TagList,
		Cover:       cr.Cover,
		UserID:      claims.UserID,
		OpenComment: cr.OpenComment,
		Status:      status,
	}).Error
	if err != nil {
		res.FailWithMsg("创建失败", c)
		return
	}

	res.OKWithMsg("创建成功", c)
}
