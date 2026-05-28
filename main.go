package main

import (
	"go-star/core"
	"go-star/flags"
	"go-star/global"

	"github.com/sirupsen/logrus"
)

func main() {
	flags.Parse()
	global.Config = core.ReadConf()
	core.InitLogrus()

	logrus.Warnf("xxx")
	logrus.Debug("yyy")
	logrus.Error("zzz")
	logrus.Info("456")
}
