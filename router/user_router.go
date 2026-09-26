package router

import (
	"go-star/api"
	"go-star/middleware"
	"time"

	"github.com/gin-gonic/gin"
)

func UserRouter(r *gin.RouterGroup) {
	app := api.App.UserApi
	r.POST("user/login", middleware.RateLimitByIP("login", 10, time.Minute), app.LoginView) // 登录
	r.POST("user/register", app.RegisterView)                                               // 注册
	r.POST("user/logout", app.LogoutView)                                                   // 退出
	r.GET("user/info", middleware.AuthMiddleware, app.UserInfoView)                         // 获取用户信息
	r.POST("user/changePwd", middleware.AuthMiddleware, app.ChangePwdView)                  // 修改密码
	r.PUT("user/update", middleware.AuthMiddleware, app.UpdateUserView)                     // 更新用户信息
	r.GET("users", middleware.AdminMiddleware, app.UserListView)                            // 获取用户列表
	r.GET("users/:id", middleware.AdminMiddleware, app.UserDetailView)                      // 获取用户详情
	r.PUT("users/:id/role", middleware.AdminMiddleware, app.UpdateRoleView)
	r.PUT("user/conf", middleware.AuthMiddleware, app.UpdateConfView) // 更新用户配置
	r.POST("users/:id/follow", middleware.AuthMiddleware, app.UserFollowView)
	r.GET("users/:id/follows", middleware.AuthMiddleware, app.UserFollowListView)
	r.GET("users/:id/fans", middleware.AuthMiddleware, app.UserFansListView)
}
