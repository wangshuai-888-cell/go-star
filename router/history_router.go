package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func HistoryRouter(r *gin.RouterGroup) {
	app := api.App.HistoryApi
	r.GET("histories", middleware.AuthMiddleware, app.HistoryListView)
	r.DELETE("histories/:id", middleware.AuthMiddleware, app.HistoryRemoveView)
	r.DELETE("histories", middleware.AuthMiddleware, app.HistoryClearView)
}
