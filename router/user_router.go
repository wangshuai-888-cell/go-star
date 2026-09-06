package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func UserRouter(r *gin.RouterGroup) {
	app := api.App.UserApi
	r.POST("user/login", app.LoginView)
	r.POST("user/register", app.RegisterView)
	r.POST("user/logout", app.LogoutView)
	r.GET("user/info", middleware.AuthMiddleware, app.UserInfoView)
	r.POST("user/changePwd", middleware.AuthMiddleware, app.ChangePwdView)
	r.PUT("user/update", middleware.AuthMiddleware, app.UpdateUserView)
	r.GET("users", middleware.AdminMiddleware, app.UserListView)
	r.GET("users/:id", middleware.AdminMiddleware, app.UserDetailView)
	r.PUT("users/:id/role", middleware.AdminMiddleware, app.UpdateRoleView)
}
