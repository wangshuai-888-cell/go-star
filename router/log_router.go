package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func LogRouter(r *gin.RouterGroup) {
	app := api.App.LogApi
	r.Use(middleware.AdminMiddleware)
	r.GET("logs", app.LogListView)
	r.GET("logs/:id", app.LogReadView)
	r.DELETE("logs", app.LogRemoveView)
}
