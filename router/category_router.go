package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func CategoryRouter(r *gin.RouterGroup) {
	app := api.App.CategoryApi
	r.POST("categories", middleware.AuthMiddleware, app.CategoryCreateView)
	r.GET("categories", middleware.AuthMiddleware, app.CategoryListView)
	r.PUT("categories/:id", middleware.AuthMiddleware, app.CategoryUpdateView)
	r.DELETE("categories/:id", middleware.AuthMiddleware, app.CategoryRemoveView)
}
