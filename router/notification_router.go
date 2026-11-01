package router

import (
	"go-star/api"
	"go-star/middleware"
	"go-star/models/enum"

	"github.com/gin-gonic/gin"
)

func NotificationRouter(r *gin.RouterGroup) {
	app := api.App.NotificationApi
	// 列表给前端展示，不要求登录
	r.GET("notifications", app.NotificationListView)
	admin := r.Group("")
	admin.Use(middleware.PermMiddleware(enum.NotificationWrite))
	admin.POST("notifications", app.NotificationCreateView)
	admin.PUT("notifications/:id", app.NotificationUpdateView)
	admin.DELETE("notifications/:id", app.NotificationRemoveView)
}
