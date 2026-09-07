package router

import (
	"go-star/api"
	"go-star/middleware"

	"github.com/gin-gonic/gin"
)

func UserRouter(r *gin.RouterGroup) {
	app := api.App.UserApi
	r.POST("user/login", app.LoginView) // 登录
	r.POST("user/register", app.RegisterView) // 注册
	r.POST("user/logout", app.LogoutView) // 退出
	r.GET("user/info", middleware.AuthMiddleware, app.UserInfoView) // 获取用户信息
	r.POST("user/changePwd", middleware.AuthMiddleware, app.ChangePwdView) // 修改密码
	r.PUT("user/update", middleware.AuthMiddleware, app.UpdateUserView) // 更新用户信息
	r.GET("users", middleware.AdminMiddleware, app.UserListView) // 获取用户列表
	r.GET("users/:id", middleware.AdminMiddleware, app.UserDetailView) // 获取用户详情
	r.PUT("users/:id/role", middleware.AdminMiddleware, app.UpdateRoleView) // 更新用户角色
}
