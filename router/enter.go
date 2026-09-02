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
	LogRouter(nr)

	addr := global.Config.System.Addr()
	r.Run(addr)
}
