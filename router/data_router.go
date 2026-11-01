package router

import (
	"go-star/api"
	"go-star/middleware"
	"go-star/models/enum"

	"github.com/gin-gonic/gin"
)

func DataRouter(r *gin.RouterGroup) {
	app := api.App.DataApi
	r.GET("data/overview", middleware.PermMiddleware(enum.DataRead), app.DataOverviewView)
}
