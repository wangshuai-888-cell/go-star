package router

import (
	"go-star/api"

	"github.com/gin-gonic/gin"
)

func UserRouter(r *gin.RouterGroup) {
	app := api.App.UserApi
	r.POST("user/login", app.LoginView)
	r.POST("user/register", app.RegisterView)
	r.POST("user/logout", app.LogoutView)
}
