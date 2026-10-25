package bloom_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-star/global"
	"go-star/utils/bloom"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

var (
	writer   *kafka.Writer
	reader   *kafka.Reader
	instance string // 本实例唯一标识，拼在group后面实现广播
	wg       sync.WaitGroup
)

type bloomMsg struct {
	ArticleID uint `json:"articleID"`
}

func Run(ctx context.Context) {
	k := global.Config.Kafka
	if len(k.Brokers) == 0 || k.BloomTopic == "" {
		logrus.Warn("未配置 kafka.bloom_topic, 布隆过滤器多实例同步不可用")
		return
	}

	writer = &kafka.Writer{
		Addr:         kafka.TCP(k.Brokers...),
		Topic:        k.BloomTopic,
		RequiredAcks: kafka.RequireOne,
	}

	// 实例唯一标识：进程ID + 启动时间，保证每个实例group不同 -> 广播模式
	instance = fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	group := k.BloomGroup + "-" + instance
	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  k.Brokers,
		Topic:    k.BloomTopic,
		GroupID:  group,
		MinBytes: 1,
		MaxBytes: 10e6,
		StartOffset: kafka.LastOffset, // 新实例从最新开始：历史由 Init 从DB加载，不重放Kafka旧消息
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		consume(ctx)
	}()
	logrus.Infof("布隆广播 kafka 已启动 topic=%s group=%s", k.BloomTopic, group)
}

func consume(ctx context.Context) {
	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				logrus.Info("布隆广播消费已停止")
				return
			}
			logrus.Errorf("布隆广播拉消息失败：%v", err)
			time.Sleep(time.Second)
			continue
		}

		var msg bloomMsg
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			logrus.Errorf("布隆广播消息不是合法 json：%v", err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		// 布隆Add是幂等的，自己发的消息自己收到，重复Add无副作用
		bloom.Add(msg.ArticleID)
		if err := reader.CommitMessages(ctx, m); err != nil {
			logrus.Errorf("布隆广播提交 offset 失败：%v", err)
			continue
		}
		logrus.Infof("布隆广播收到新文章 id =%d offset=%d", msg.ArticleID, m.Offset)
	}
}

// 新文章创建后广播给所有实例
func Broadcast(articleID uint) {
	bloom.Add(articleID) // 本地先加，不等消息绕回来
	logrus.Infof("布隆本地添加文章 id=%d", articleID)
	if writer == nil {
		return // Kafka没启用：只在本地生效，单实例场景够用
	}
	// 把结构体转为json字符串
	b, err := json.Marshal(bloomMsg{ArticleID: articleID})
	if err != nil {
		logrus.Errorf("布隆广播序列化失败：%v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := writer.WriteMessages(ctx, kafka.Message{Value: b}); err != nil {
		logrus.Errorf("布隆广播投递 kafka 失败：%v", err) // 广播失败只影响其他实例，本地已加
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
	logrus.Info("布隆广播 kafka 已关闭")
}
