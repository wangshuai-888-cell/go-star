package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func SearchRouter(r *gin.RouterGroup) {
	app := api.App.SearchApi
	r.POST("search/logs", middleware.AuthMiddleware, app.SearchLogView)
	r.GET("search/hot", app.SearchHotView) // 热榜可不登录，方便测 EXPLAIN
	r.GET("search/history", middleware.AuthMiddleware, app.SearchHistoryView)
}
