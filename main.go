package main

import (
	"go-star/core"
	"go-star/flags"
	"go-star/global"
	"go-star/router"
)

func main() {
	flags.Parse()                   // 读取配置文件是否存在还有命令行参数
	global.Config = core.ReadConf() // 将配置文件读取到全局变量中
	core.InitLogrus()               // 打印日志
	global.DB = core.InitDB()       // 连接数据库
	global.Redis = core.InitRedis() // 连接redis

	flags.Run() // 根据运行命令参数，决定是否对数据库进行迁移

	router.Run() // 启动gin服务
}
