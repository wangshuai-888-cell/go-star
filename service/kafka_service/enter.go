package kafka_service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"go-star/global"
	"go-star/models"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type SearchEvent struct {
	EventID    string    `json:"eventID"`
	UserID     uint      `json:"userID"`
	Keyword    string    `json:"keyword"`
	SearchedAt time.Time `json:"searchedAt"`
}

var (
	writer *kafka.Writer
	reader *kafka.Reader
	wg     sync.WaitGroup
)

func Run(ctx context.Context) {
	k := global.Config.Kafka
	if len(k.Brokers) == 0 || k.Topic == "" {
		logrus.Warn("未配置 kafka，跳过生产消费")
		return
	}

	writer = &kafka.Writer{
		Addr:     kafka.TCP(k.Brokers...),
		Topic:    k.Topic,
		Balancer: &kafka.LeastBytes{}, // 决定消息发送到哪个分区，LeastBytes：哪个分区数据量最少就发送到哪个分区
	}

	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  k.Brokers, // 连接kafka的地址
		Topic:    k.Topic,   // 主题
		GroupID:  k.Group,   // 消费组
		MinBytes: 1,         // 最小读取消息大小
		MaxBytes: 10e6,      // 最大读取消息大小
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		consume(ctx)
	}()
	logrus.Infof("kafka 已启动 topic=%s group=%s", k.Topic, k.Group)
}

func consume(ctx context.Context) {
	for {
		m, err := reader.FetchMessage(ctx) // 拉取消息
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				logrus.Info("kafka 消费已停止")
				return
			}
			logrus.Errorf("kafka 拉消息失败: %v", err)
			time.Sleep(time.Second) // 休眠1秒，避免频繁重试，不然会一直刷屏
			continue
		}

		var ev SearchEvent
		if err := json.Unmarshal(m.Value, &ev); err != nil {
			logrus.Errorf("kafka 消息不是合法 json: %v", err)
			_ = reader.CommitMessages(ctx, m) // CommitMessages是告诉kafka，这个消费组已经处理完这条消息了，下次不要再发给我了
			// 如果消息解析失败，也要提交，不然会一直卡在这里，后面合法的也会被堵死
			continue
		}

		if err := persistSearch(ev); err != nil {
			logrus.Errorf("消费搜索事件失败: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if err := reader.CommitMessages(ctx, m); err != nil {
			logrus.Errorf("kafka 提交 offset 失败: %v", err)
			continue
		}
		logrus.Infof("已消费搜索 eventID=%s keyword=%s userID=%d partition=%d offset=%d",
			ev.EventID, ev.Keyword, ev.UserID, m.Partition, m.Offset)
	}
}

func persistSearch(ev SearchEvent) error {
	if ev.EventID == "" {
		return fmt.Errorf("缺少eventID，拒绝消费")
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&models.SearchEventModel{
			EventID: ev.EventID,
			UserID:  ev.UserID,
			Keyword: ev.Keyword,
		}).Error
		if err != nil {
			if isDuplicateKey(err) {
				logrus.Infof("搜索事件已处理过了，跳过eventID=%s", ev.EventID)
				return nil
			}
			// 如果有其他错误，则返回错误
			return fmt.Errorf("记录搜索事件 eventID=%s: %w", ev.EventID, err)
		}

		var hot models.SearchHotModel
		err = tx.Where("keyword = ?", ev.Keyword).Take(&hot).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询热词 keyword=%s:%w", ev.Keyword, err)
			}
			hot = models.SearchHotModel{
				Keyword:      ev.Keyword,
				SearchCount:  1,
				LastSearchAt: ev.SearchedAt,
			}
			if err := tx.Create(&hot).Error; err != nil {
				return fmt.Errorf("创建热词 keyword=%s: %w", ev.Keyword, err)
			}
		} else {
			if err := tx.Model(&hot).Updates(map[string]any{
				"search_count":   gorm.Expr("search_count + ?", 1),
				"last_search_at": ev.SearchedAt,
			}).Error; err != nil {
				return fmt.Errorf("更新热词 keyword=%s: %w", ev.Keyword, err)
			}
		}
		if err := tx.Create(&models.SearchHistoryModel{
			UserID:     ev.UserID,
			Keyword:    ev.Keyword,
			SearchedAt: ev.SearchedAt,
		}).Error; err != nil {
			return fmt.Errorf("写入搜索历史 userID=%d keyword=%s: %w", ev.UserID, ev.Keyword, err)
		}
		return nil
	})
}

// EventID有唯一索引，在表中插入数据时会根据这个来判断是否重复
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "1062")
}

func ProduceSearch(ev SearchEvent) error {
	if writer == nil {
		return errors.New("kafka writer 未启动")
	}
	if ev.EventID == "" {
		ev.EventID = uuid.New().String()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.Keyword),
		Value: b,
	})
}

func Shutdown() {
	if reader != nil {
		_ = reader.Close()
	}
	if writer != nil {
		_ = writer.Close()
	}
	wg.Wait()
	logrus.Info("kafka 已关闭")
}
