package router

import (
	"go-star/global"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func Run() {
	gin.SetMode(global.Config.System.GinMode) // 设置gin的模式，读取的是settings.yaml文件中的gin_mode
	r := gin.Default() // 创建gin实例

	r.Static("/uploads", "uploads")

	nr := r.Group("/api")

	nr.Use(middleware.LogMiddleware)
	SiteRouter(nr)
	UserRouter(nr)

	// 日志接口需要管理员权限，在入口显式挂中间件，避免污染整个 /api 组
	logGroup := nr.Group("")
	logGroup.Use(middleware.AdminMiddleware)
	LogRouter(logGroup)

	addr := global.Config.System.Addr()
	r.Run(addr)
}
