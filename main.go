package main

import (
	"context"
	"go-star/core"
	"go-star/flags"
	"go-star/global"
	"go-star/router"
	"go-star/service/cron_service"
	"go-star/service/message_service"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	flags.Parse()                   // 读取配置文件是否存在还有命令行参数
	global.Config = core.ReadConf() // 将配置文件读取到全局变量中
	core.InitLogrus()               // 打印日志
	global.DB = core.InitDB()       // 连接数据库
	global.Redis = core.InitRedis() // 连接redis
	core.InitKafka()                // 探测 kafka
	core.InitIPDB()                 // 加载 IP 地址库
	flags.Run()                     // 根据运行命令参数，决定是否对数据库进行迁移

	// 监听ctrl+c或者kill信号，触发后ctx会被取消
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cron_service.Run(ctx) // 启动定时发布任务
	message_service.Run() //启动站内信异步队列

	router.Run(ctx)            // 启动gin服务
	message_service.Shutdown() // 关闭站内信异步队列
}
