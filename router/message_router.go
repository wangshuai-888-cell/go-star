package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func MessageRouter(r *gin.RouterGroup) {
	app := api.App.MessageApi
	r.GET("messages", middleware.AuthMiddleware, app.MessageListView)
	r.GET("messages/unread", middleware.AuthMiddleware, app.MessageUnreadCountView)
	r.POST("messages/:id/read", middleware.AuthMiddleware, app.MessageReadView)
	r.POST("messages/read-all", middleware.AuthMiddleware, app.MessageReadAllView)
}
