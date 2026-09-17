package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func ImageRouter(r *gin.RouterGroup) {
	app := api.App.ImageApi
	r.POST("images", middleware.AuthMiddleware, app.ImageUploadView)
	r.GET("images", middleware.AuthMiddleware, app.ImageListView)
	r.DELETE("images/:id", middleware.AuthMiddleware, app.ImageRemoveView)
}
