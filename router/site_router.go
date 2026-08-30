package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func SiteRouter(r *gin.RouterGroup) {
	app := api.App.SiteApi
	r.GET("site", app.SiteInfoView)
	r.PUT("site", middleware.AuthMiddleware, app.SiteUpdateView)
}
