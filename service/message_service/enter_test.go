package message_service

import (
	"go-star/models"
	"sync"
	"testing"
)

func setupMemPersist(t *testing.T) *[]models.UserMessageModel {
	t.Helper()
	var mu sync.Mutex
	got := make([]models.UserMessageModel, 0)

	old := persist
	persist = func(msg models.UserMessageModel) error {
		mu.Lock()
		got = append(got, msg)
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() {
		persist = old
	})
	return &got
}

func TestPushWithoutKafkaWritesDirectly(t *testing.T) {
	got := setupMemPersist(t)
	writer = nil // 确保未启动 kafka
	Push(models.UserMessageModel{Title: "无kafka时同步写", RevUserID: 1})
	if len(*got) != 1 {
		t.Fatalf("应同步写入 1 条，实际 %d", len(*got))
	}
}
