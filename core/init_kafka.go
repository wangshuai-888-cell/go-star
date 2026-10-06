package core

import (
	"go-star/global"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// InitKafka 探测线上/配置的 Kafka 是否可达（暂不启动生产消费）
func InitKafka() {
	k := global.Config.Kafka
	if len(k.Brokers) == 0 || k.Brokers[0] == "" {
		logrus.Warn("未配置 kafka.brokers，跳过连接")
		return
	}

	dialer := &kafka.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", k.Brokers[0])
	if err != nil {
		logrus.Errorf("连接kafka失败 %s", err)
		return
	}
	defer conn.Close()

	brokers, err := conn.Brokers()
	if err != nil {
		logrus.Errorf("获取kafka broker列表失败 %s", err)
		return
	}
	logrus.Infof("连接kafka成功 addr=%s broker数=%d topic=%s", k.Brokers[0], len(brokers), k.Topic)
}
