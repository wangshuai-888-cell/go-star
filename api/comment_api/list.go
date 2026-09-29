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

type CommentUser struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type CommentListItem struct {
	models.CommentModel
	User           CommentUser       `json:"user"`
	SubCommentList []CommentListItem `json:"subCommentList"`
}

func toCommentItem(c models.CommentModel) CommentListItem {
	item := CommentListItem{
		CommentModel: c,
		User: CommentUser{
			ID:       c.UserModel.ID,
			Nickname: c.UserModel.Nickname,
			Avatar:   c.UserModel.Avatar,
		},
		SubCommentList: make([]CommentListItem, 0),
	}
	item.CommentModel.SubCommentList = nil // 避免和下面的 VO 重复
	for _, sub := range c.SubCommentList {
		if sub != nil {
			item.SubCommentList = append(item.SubCommentList, toCommentItem(*sub))
		}
	}
	return item
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
		PageInfo: cr.PageInfo,
		Where:    query,
		Preloads: []string{
			"UserModel",
			"SubCommentList",
			"SubCommentList.UserModel",
		},
		DefaultOrder: "created_at desc",
	})
	if err != nil {
		res.FailWithMsg("获取评论列表失败", c)
		return
	}

	out := make([]CommentListItem, 0, len(list))
	for _, item := range list {
		out = append(out, toCommentItem(item))
	}
	res.OKWithList(out, count, c)
}
