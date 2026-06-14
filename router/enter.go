package router

import (
	"go-star/global"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func Run() {
	r := gin.Default()

	nr := r.Group("/api")

	nr.Use(middleware.LogMiddleware)
	SiteRouter(nr)

	addr := global.Config.System.Addr()
	r.Run(addr)
}
