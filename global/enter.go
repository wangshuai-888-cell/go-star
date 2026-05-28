package global

import (
	"go-star/conf"

	"gorm.io/gorm"
)

var (
	Config *conf.Config
	DB     *gorm.DB
)
