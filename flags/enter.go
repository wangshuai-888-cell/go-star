package flags

import (
	"flag"
	"os"
)

type Options struct {
	File    string
	DB      bool
	Version bool
}

var FlagOptions = new(Options) // 创建指针，方便在其他文件中读取该值

// Parse 解析命令函参数
func Parse() {
	flag.StringVar(&FlagOptions.File, "f", "settings.yaml", "配置文件")
	flag.BoolVar(&FlagOptions.DB, "db", false, "数据库迁移")
	flag.BoolVar(&FlagOptions.Version, "v", false, "版本")
	flag.Parse()

}

func Run() {
	if FlagOptions.DB {
		FlagDB()
		os.Exit(0) // 程序退出
	}
}
