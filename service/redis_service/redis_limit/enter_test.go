package redis_limit

import (
	"testing"
	"time"

	"go-star/global"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
)

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	global.Redis = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	t.Cleanup(func() {
		global.Redis = nil
		mr.Close()
	})
	return mr
}

func TestAllow(t *testing.T) {
	setupTestRedis(t)

	// 每分钟最多 2 次
	if !Allow("login", "127.0.0.1", 2, time.Minute) {
		t.Fatal("第1次应放行")
	}
	if !Allow("login", "127.0.0.1", 2, time.Minute) {
		t.Fatal("第2次应放行")
	}
	if Allow("login", "127.0.0.1", 2, time.Minute) {
		t.Fatal("第3次应拒绝")
	}
}

func TestAllow_DifferentIdentity(t *testing.T) {
	setupTestRedis(t)

	if !Allow("login", "1.1.1.1", 1, time.Minute) {
		t.Fatal("IP A 第1次应放行")
	}
	if Allow("login", "1.1.1.1", 1, time.Minute) {
		t.Fatal("IP A 第2次应拒绝")
	}
	// 换一个 IP，计数独立
	if !Allow("login", "2.2.2.2", 1, time.Minute) {
		t.Fatal("IP B 第1次应放行")
	}
}

func TestAllow_RedisNil(t *testing.T) {
	global.Redis = nil
	if !Allow("login", "x", 1, time.Minute) {
		t.Fatal("Redis 为 nil 时应放行")
	}
}
