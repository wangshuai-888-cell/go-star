package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func CollectRouter(r *gin.RouterGroup) {
	app := api.App.CollectApi
	r.POST("collects", middleware.AuthMiddleware, app.CollectCreateView)
	r.GET("collects", middleware.AuthMiddleware, app.CollectListView)
	r.PUT("collects/:id", middleware.AuthMiddleware, app.CollectUpdateView)
	r.DELETE("collects/:id", middleware.AuthMiddleware, app.CollectRemoveView)
	r.POST("articles/:id/collect", middleware.AuthMiddleware, app.ArticleCollectView)     // 文章保存进收藏夹
	r.GET("collects/:id/articles", middleware.AuthMiddleware, app.CollectArticleListView) // 收藏夹文章列表
}
