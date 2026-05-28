package main

import (
	"go-star/core"
	"go-star/flags"
	"go-star/global"
)

func main() {
	flags.Parse()
	global.Config = core.ReadConf()
	core.InitLogrus()
	global.DB = core.InitDB()
}
