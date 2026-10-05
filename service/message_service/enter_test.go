package message_service

import (
	"go-star/models"
	"sync"
	"testing"
	"time"
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

func TestPushAndShutdownDrains(t *testing.T) {
	got := setupMemPersist(t)
	Run()
	Push(models.UserMessageModel{Title: "关注1", RevUserID: 4})
	Push(models.UserMessageModel{Title: "关注2", RevUserID: 5})
	Shutdown()
	if len(*got) != 2 {
		t.Fatalf("排空后应写入 2 条，实际 %d", len(*got))
	}
}

func TestPushAfterShutdownWritesDirectly(t *testing.T) {
	got := setupMemPersist(t)

	Run()
	Shutdown()

	Push(models.UserMessageModel{Title: "停机后仍要留下"})
	time.Sleep(20 * time.Millisecond)

	if len(*got) != 1 {
		t.Fatalf("停机后 Push 应同步写入，实际 %d", len(*got))
	}
	if (*got)[0].Title != "停机后仍要留下" {
		t.Errorf("title = %s", (*got)[0].Title)
	}
}
