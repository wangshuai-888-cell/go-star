package api

import (
	"go-star/api/article_api"
	categoryapi "go-star/api/category_api"
	"go-star/api/log_api"
	"go-star/api/site_api"
	"go-star/api/user_api"
)

type Api struct {
	SiteApi     site_api.SiteApi
	LogApi      log_api.LogApi
	UserApi     user_api.UserApi
	CategoryApi categoryapi.CategoryApi
	ArticleApi  article_api.ArticleApi
}

var App = Api{}
