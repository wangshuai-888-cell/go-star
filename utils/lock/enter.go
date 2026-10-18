package lock

import (
	"fmt"
	"go-star/global"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

/*
*
Acquire 基于 Redis SET NX EX 实现分布式锁
name：锁的名字（不同任务用不同的锁）
expire：锁的过期时间，必须大于任务的最长执行时间
返回ok表示抢到锁，release是释放函数（任务完成后必须调用）
Redis不可用时降级为直接放行：宁可重复执行，也不能让任务停摆
*/
func Acquire(name string, expire time.Duration) (ok bool, release func()) {
	// 降级：redis不可用直接放行
	if global.Redis == nil {
		return true, func() {}
	}
	token := uuid.NewString()
	key := fmt.Sprintf("lock:%s", name)
	// NX + EX 一条命令原子完成，防止进程崩溃导致死锁
	// SetNX的本质：在redis中写入一个key-value对，
	ok, err := global.Redis.SetNX(key, token, expire).Result()
	if err != nil {
		logrus.Errorf("获取分布式锁失败 %s: %v", name, err)
		return true, func() {} // Redis故障也降级放行
	}
	if !ok {
		return false, func() {} // 别人持有锁，本轮跳过
	}
	// 释放：Lua保证“校验token+删除”原子，防止误删别人的锁
	release = func() {
		script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
		`
		_, _ = global.Redis.Eval(script, []string{key}, token).Result()
	}
	return true, release
}
