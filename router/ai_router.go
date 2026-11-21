package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func AiRouter(r *gin.RouterGroup) {
	app := api.App.AiApi
	g := r.Group("ai")
	g.Use(middleware.AuthMiddleware)
	g.POST("chat", app.ChatStreamView)
	g.POST("ask", app.AskStreamView)
}
