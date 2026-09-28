package router

import (
	"go-star/api"

	"github.com/gin-gonic/gin"
)

func DataRouter(r *gin.RouterGroup) {
	app := api.App.DataApi
	r.GET("data/overview", app.DataOverviewView)
}
