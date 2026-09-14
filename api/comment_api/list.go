package comment_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"

	"github.com/gin-gonic/gin"
)

type CommentListRequest struct {
	common.PageInfo
}

func (CommentApi) CommentListView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}
	var cr CommentListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	var article models.ArticleModel
	if err := global.DB.Take(&article, idCr.ID).Error; err != nil {
		res.FailWithMsg("文章不存在", c)
		return
	}

	query := global.DB.Where("article_id = ? AND parent_id IS NULL", article.ID)
	list, count, err := common.ListQuery(models.CommentModel{}, common.Options{
		PageInfo:     cr.PageInfo,
		Where:        query,
		Preloads:     []string{"SubCommentList"}, // 其实就是相当于["SubCommentList"]
		DefaultOrder: "created_at desc",          // 按照created_at创建时间节点从新到旧排列
	})
	if err != nil {
		res.FailWithMsg("获取评论列表失败", c)
		return
	}
	res.OKWithList(list, count, c)
}
