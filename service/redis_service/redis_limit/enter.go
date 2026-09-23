package redis_limit

import (
	"fmt"
	"go-star/global"
	"time"
)

// Allow 返回 true，表示放行，window内最多max次
// action代表动作，用来区分限制的是哪件事，比如登录、评论等
// identity代表身份，可以是IP，也可以是用户ID
func Allow(action, identity string, max int64, window time.Duration) bool {
	if global.Redis == nil {
		return true // redis挂了先放行，避免全站打不开
	}
	key := fmt.Sprintf("rate:%s:%s", action, identity)
	n, err := global.Redis.Incr(key).Result() //global.Redis.Incr：对key的值加1
	if err != nil {
		return true
	}
	if n == 1 {
		_ = global.Redis.Expire(key, window).Err() //global.Redis.Expire：设置key的过期时间
	}
	return n <= max
}
