package router

import (
	"go-star/global"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func Run() {
	gin.SetMode(global.Config.System.GinMode)
	r := gin.Default()

	r.Static("/uploads", "uploads")

	nr := r.Group("/api")

	nr.Use(middleware.LogMiddleware)
	SiteRouter(nr)
	LogRouter(nr)

	addr := global.Config.System.Addr()
	r.Run(addr)
}
