package message_service

import (
	"context"
	"go-star/global"
	"go-star/models"

	"github.com/sirupsen/logrus"
)

var ch chan models.UserMessageModel

const (
	workerCount = 2   // 同时写库的数量
	queueSize   = 100 // 队列的缓冲
)

func Run(ctx context.Context) {
	ch = make(chan models.UserMessageModel, queueSize)

	for i := 0; i < workerCount; i++ {
		id := i + 1
		go func() {
			for {
				select {
				case <-ctx.Done():
					logrus.Infof("消息worker-%d 已停止", id)
					return
				case msg := <-ch:
					if err := global.DB.Create(&msg).Error; err != nil {
						logrus.Errorf("异步写消息失败: %v", err)
					}
				}
			}
		}()
	}
	logrus.Infof("消息异步队列已启动 worker=%d queue=%d", workerCount, queueSize)
}

// 业务方调用：把消息丢进队列
func Push(msg models.UserMessageModel) {
	if ch == nil {
		_ = global.DB.Create(&msg).Error
		return
	}

	select {
	case ch <- msg:
		// 丢进去了，接口不用等写库
	default:
		// 队列满了：降级同步写，尽量不丢消息
		logrus.Warn("消息队列已满，降级为同步写入")
		_ = global.DB.Create(&msg).Error
	}
}
