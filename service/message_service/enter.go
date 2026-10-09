package message_service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"go-star/global"
	"go-star/models"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

var (
	writer *kafka.Writer
	reader *kafka.Reader
	wg     sync.WaitGroup
)

// 单测可替换；Kafka 消费和降级写库都走它
var persist = func(msg models.UserMessageModel) error {
	if global.DB == nil {
		return nil
	}
	return global.DB.Create(&msg).Error
}

func Run(ctx context.Context) {
	k := global.Config.Kafka
	if len(k.Brokers) == 0 || k.MessageTopic == "" {
		logrus.Warn("未配置 kafka.message_topic，站内信将同步写库")
		return
	}

	writer = &kafka.Writer{
		Addr:         kafka.TCP(k.Brokers...),
		Topic:        k.MessageTopic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
	}

	group := k.MessageGroup
	if group == "" {
		group = k.Group + "-message"
	}
	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  k.Brokers,
		Topic:    k.MessageTopic,
		GroupID:  group,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	wg.Add(1)
	go func() {
		defer wg.Done() // defer：函数退出前执行该函数，所以consume返回return时就会执行wg.Done()
		consume(ctx)
	}()
	logrus.Infof("站内信 kafka 已启动 topic=%s group=%s", k.MessageTopic, group)
}

func consume(ctx context.Context) {
	for {
		// reader.FetchMessage：读取信息，ctx：上下文，用于取消阻塞操作，如超时等
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				logrus.Info("站内信 kafka 消费已停止")
				return
			}
			logrus.Errorf("站内信拉消息失败: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var msg models.UserMessageModel
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			logrus.Errorf("站内信消息不是合法 json: %v", err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		if err := persist(msg); err != nil {
			logrus.Errorf("站内信写库失败: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := reader.CommitMessages(ctx, m); err != nil {
			logrus.Errorf("站内信提交 offset 失败: %v", err)
			continue
		}
		logrus.Infof("已消费站内信 revUserID=%d type=%d title=%s offset=%d",
			msg.RevUserID, msg.Type, msg.Title, m.Offset)
	}
}

func Push(msg models.UserMessageModel) {
	if writer == nil {
		if err := persist(msg); err != nil {
			logrus.Errorf("站内信同步写入失败: %v", err)
		}
		return
	}

	// 可选：带上 eventID，以后做幂等用；库表暂不强制
	b, err := json.Marshal(msg)
	if err != nil {
		logrus.Errorf("站内信序列化失败: %v", err)
		_ = persist(msg)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatUint(uint64(msg.RevUserID), 10)),
		Value: b,
	})
	if err != nil {
		logrus.Errorf("站内信投递 kafka 失败，降级写库: %v", err)
		_ = persist(msg)
	}
}

func Shutdown() {
	if reader != nil {
		_ = reader.Close()
		reader = nil
	}
	if writer != nil {
		_ = writer.Close()
		writer = nil
	}
	wg.Wait()
	logrus.Info("站内信 kafka 已关闭")
}
