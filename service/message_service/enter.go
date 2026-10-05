package message_service

import (
	"go-star/global"
	"go-star/models"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

var ch chan models.UserMessageModel
var wg sync.WaitGroup
var stopped atomic.Bool // atomic.Bool 是原子操作的布尔值

var persist = func(msg models.UserMessageModel) error {
	if global.DB == nil {
		return nil
	}
	return global.DB.Create(&msg).Error
}

const (
	workerCount = 2
	queueSize   = 100
)

func Run() {
	stopped.Store(false)
	ch = make(chan models.UserMessageModel, queueSize)

	for i := 0; i < workerCount; i++ {
		id := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done() // 每结束一个协程，等待组计数器减1
			// 管道关闭且取完后，range 自动结束
			for msg := range ch {
				if err := persist(msg); err != nil {
					logrus.Errorf("异步写消息失败: %v", err)
				}
			}
			logrus.Infof("消息worker-%d 已停止（队列已排空）", id)
		}()
	}
	logrus.Infof("消息异步队列已启动 worker=%d queue=%d", workerCount, queueSize)
}

// HTTP 关完后再调：关闭管道并等待 worker 把剩余消息写完
func Shutdown() {
	if ch == nil {
		return
	}
	stopped.Store(true)
	close(ch)
	wg.Wait()
	ch = nil
	logrus.Info("消息队列已排空")
}

func Push(msg models.UserMessageModel) {
	// 如果要停机了，则直接手动写入数据库
	if ch == nil || stopped.Load() { // stopped.Load():读出stopped当前的值
		_ = persist(msg)
		return
	}

	select {
	case ch <- msg: // 如果还能塞得下，就说明队列还有空位
	default:
		logrus.Warn("消息队列已满，降级为同步写入")
		_ = persist(msg)
	}
}
