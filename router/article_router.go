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
	r.GET("articles/:id", middleware.AuthMiddleware, app.ArticleDetailView)
	r.PUT("articles/:id", middleware.AuthMiddleware, app.ArticleUpdateView)
	r.DELETE("articles/:id", middleware.AuthMiddleware, app.ArticleRemoveView)
}
