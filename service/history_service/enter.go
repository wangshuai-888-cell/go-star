package history_service

import (
	"context"
	"go-star/global"
	"go-star/models"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	queueSize   = 4096            // 队列容量：瞬时高峰最多缓冲这么多条
	workerCount = 2               // 消费者数量
	batchSize   = 100             // 攒够100条就写库
	flushEvery  = 5 * time.Second // 或者最多等5秒
)

// 一条待写入的浏览记录
type lookItem struct {
	UserID    uint
	ArticleID uint
}

var (
	queue chan lookItem // 用来传输lookItem类型的管道
	wg    sync.WaitGroup
)

// Run 启动浏览历史异步写入（main 里调用）
func Run(_ context.Context) {
	queue = make(chan lookItem, queueSize)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	logrus.Infof("浏览历史异步写入已启动：worker=%d 队列=%d 批量=%d 间隔=%s",
		workerCount, queueSize, batchSize, flushEvery)
}

// Add 接口层调用：只入队，立即返回；队列满则丢弃（浏览历史可容忍丢失）
func Add(userID, articleID uint) {
	if queue == nil {
		return
	}
	select {
	case queue <- lookItem{UserID: userID, ArticleID: articleID}:
	default:
		logrus.Warnf("浏览历史队列已满，丢弃 user=%d article=%d", userID, articleID)
	}
}

// worker 消费队列，攒批写库
func worker() {
	defer wg.Done()

	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()

	batch := make([]lookItem, 0, batchSize)
	for {
		select {
		case item, ok := <-queue:
			if !ok {
				// 队列已关闭：把剩下的写完再退出
				flush(&batch)
				return
			}
			batch = append(batch, item)
			if len(batch) >= batchSize {
				flush(&batch)
			}
		case <-ticker.C:
			// 量少的时候也要保证5秒内落库一次
			flush(&batch)
		}
	}
}

// flush 批量写入：已存在则只刷新 updated_at
func flush(batch *[]lookItem) {
	if len(*batch) == 0 {
		return
	}

	list := make([]models.UserArticleLookHistoryModel, 0, len(*batch))
	for _, item := range *batch {
		list = append(list, models.UserArticleLookHistoryModel{
			UserID:    item.UserID,
			ArticleID: item.ArticleID,
		})
	}

	err := global.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "article_id"}},
		DoUpdates: clause.Assignments(map[string]any{"updated_at": gorm.Expr("NOW()")}),
	}).CreateInBatches(&list, batchSize).Error

	if err != nil {
		logrus.Errorf("浏览历史批量写入失败 %d 条：%v", len(list), err)
	} else {
		logrus.Infof("浏览历史批量写入 %d 条", len(list))
	}
	*batch = (*batch)[:0]
}

// Shutdown: 优雅关闭：关闭列表，等所有worker把剩余数据刷完
func Shutdown() {
	if queue == nil {
		return
	}
	close(queue)
	wg.Wait()
}
