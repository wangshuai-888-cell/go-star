package api

import (
	"go-star/api/article_api"
	categoryapi "go-star/api/category_api"
	"go-star/api/collect_api"
	"go-star/api/comment_api"
	"go-star/api/history_api"
	"go-star/api/image_api"
	"go-star/api/log_api"
	"go-star/api/notification_api"
	"go-star/api/user_api"
)

type Api struct {
	LogApi          log_api.LogApi
	UserApi         user_api.UserApi
	CategoryApi     categoryapi.CategoryApi
	ArticleApi      article_api.ArticleApi
	ImageApi        image_api.ImageApi
	CommentApi      comment_api.CommentApi
	CollectApi      collect_api.CollectApi
	HistoryApi      history_api.HistoryApi
	NotificationApi notification_api.NotificationApi
}

var App = Api{}
