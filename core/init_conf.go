package core

import (
	"fmt"
	"go-star/conf"
	"go-star/flags"
	"os"

	"gopkg.in/yaml.v2"
)

type System struct {
	Ip   string `yaml:"ip"`
	Port string `yaml:"port"`
}
type Config struct {
	System System `yaml:"system"`
}

func ReadConf() (c *conf.Config) {
	byteData, err := os.ReadFile(flags.FlagOptions.File)
	if err != nil {
		panic(err)
	}
	c = new(conf.Config)
	err = yaml.Unmarshal(byteData, c)
	if err != nil {
		panic(fmt.Sprintf("yaml文件格式错误:%s", err))
	}

	fmt.Printf("读取配置文件 %s 成功\n", flags.FlagOptions.File)
	return
}
