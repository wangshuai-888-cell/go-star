package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func SiteRouter(r *gin.RouterGroup) {
	app := api.App.SiteApi
	r.GET("site", app.SiteInfoView)
	// middleware.AuthMiddleware是中间件，用于验证用户是否登录
	r.PUT("site", middleware.AuthMiddleware, app.SiteUpdateView)
}
