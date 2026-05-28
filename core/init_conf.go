package core

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var confPath = "settings.yaml"

type System struct {
	Ip   string `yaml:"ip"`
	Port string `yaml:"port"`
}
type Config struct {
	System System `yaml:"system"`
}

func ReadConf() {
	byteData, err := os.ReadFile(confPath)
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(byteData, &config)
	if err != nil {
		panic(fmt.Sprintf("yaml文件格式错误:%s", err))
	}
	fmt.Println(config)
}
