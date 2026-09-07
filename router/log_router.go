package router

import (
	"go-star/api"

	"github.com/gin-gonic/gin"
)

func LogRouter(r *gin.RouterGroup) {
	app := api.App.LogApi
	r.GET("logs", app.LogListView) // 获取日志列表
	r.GET("logs/:id", app.LogReadView) // 获取日志详情
	r.DELETE("logs", app.LogRemoveView) // 删除日志
}
