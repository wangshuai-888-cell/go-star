package router

import (
	"go-star/api"
	"go-star/middleware"
	"go-star/models/enum"

	"github.com/gin-gonic/gin"
)

func BannerRouter(r *gin.RouterGroup) {
	app := api.App.BannerApi
	r.GET("banners", app.BannerListView)

	admin := r.Group("")
	admin.Use(middleware.PermMiddleware(enum.BannerWrite))
	admin.POST("banners", app.BannerCreateView)
	admin.PUT("banners/:id", app.BannerUpdateView)
	admin.DELETE("banners/:id", app.BannerRemoveView)
}
