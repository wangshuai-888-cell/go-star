package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func ArticleRouter(r *gin.RouterGroup) {
	app := api.App.ArticleApi
	r.POST("articles", middleware.AuthMiddleware, app.ArticleCreateView)
	r.GET("articles", middleware.AuthMiddleware, app.ArticleListView)
	r.GET("articles/hot", middleware.AuthMiddleware, app.ArticleHotListView) // 如果写在articles/:id后面，hot会被当做id
	r.GET("articles/:id", middleware.AuthMiddleware, app.ArticleDetailView)
	r.PUT("articles/:id", middleware.AuthMiddleware, app.ArticleUpdateView)
	r.DELETE("articles/:id", middleware.AuthMiddleware, app.ArticleRemoveView)
	r.POST("articles/:id/digg", middleware.AuthMiddleware, app.ArticleDiggView)
	r.POST("articles/:id/top", middleware.AuthMiddleware, app.ArticleTopView)
	r.GET("users/:id/top-articles", middleware.AuthMiddleware, app.UserTopArticleListView)
}
