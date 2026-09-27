package enum

type ArticleStatus int8

const (
	ArticleStatusDraft     ArticleStatus = 1 // 草稿
	ArticleStatusReview    ArticleStatus = 2 // 审核中
	ArticleStatusPublished ArticleStatus = 3 // 已发布
	ArticleStatusRejected  ArticleStatus = 4 // 已驳回
)
