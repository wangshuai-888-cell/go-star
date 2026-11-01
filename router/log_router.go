package router

import (
	"go-star/api"
	"go-star/middleware"
	"go-star/models/enum"

	"github.com/gin-gonic/gin"
)

func LogRouter(r *gin.RouterGroup) {
	app := api.App.LogApi
	r.GET("logs", middleware.PermMiddleware(enum.LogRead), app.LogListView)     // 获取日志列表
	r.GET("logs/:id", middleware.PermMiddleware(enum.LogRead), app.LogReadView) // 获取日志详情
	// 职责分离：审计日志不允许被被审计者销毁，log:delete 不授予任何角色
	r.DELETE("logs", middleware.PermMiddleware(enum.LogDelete), app.LogRemoveView) // 删除日志
}
