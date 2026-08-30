package core

import (
	"go-star/global"

	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
)

func InitRedis() *redis.Client {
	r := global.Config.Redis
	redisDB := redis.NewClient(&redis.Options{
		Addr:     r.Addr,     // 默认值
		Password: r.Password, // 密码
		DB:       r.DB,
	})
	_, err := redisDB.Ping().Result()
	if err != nil {
		logrus.Errorf("连接redis失败 %s", err)
	}
	logrus.Info("连接redis成功")
	return redisDB
}
