package router

import (
	"context"
	"go-star/global"
	"go-star/middleware"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Run(ctx context.Context) {
	gin.SetMode(global.Config.System.GinMode) // 设置 gin 模式，来自 -f 指定的配置文件
	r := gin.Default()                        // 创建gin实例

	r.Static("/uploads", "uploads")

	nr := r.Group("/api")

	nr.Use(middleware.LogMiddleware)
	UserRouter(nr)
	CategoryRouter(nr)
	ArticleRouter(nr)
	ImageRouter(nr)
	CommentRouter(nr)
	CollectRouter(nr)
	HistoryRouter(nr)
	NotificationRouter(nr)
	BannerRouter(nr)
	MessageRouter(nr)

	// 日志接口需要管理员权限，在入口显式挂中间件，避免污染整个 /api 组
	logGroup := nr.Group("")
	logGroup.Use(middleware.AdminMiddleware)
	LogRouter(logGroup)
	DataRouter(logGroup)

	addr := global.Config.System.Addr()
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// HTTP 在后台监听请求，主协程等待停机信号
	go func() {
		logrus.Infof("HTTP 服务启动，监听地址：%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HTTP 服务启动失败：%v", err)
		}
	}()

	<-ctx.Done() // 等待停机信号
	logrus.Info("收到停机信号，开始关闭HTTP服务")

	// 最多再等5秒，让进程中的请求收尾
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logrus.Errorf("HTTP 服务关闭失败：%v", err)
		return
	}
	logrus.Info("HTTP 服务已关闭")
}
