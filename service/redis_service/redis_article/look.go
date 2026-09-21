package redis_article

import (
	"errors"
	"fmt"
	"go-star/global"
	"strconv"
)

// 浏览数攒够10次后，再一次性写入数据库
const lookFlushStep = 10

func lookKey(id uint) string {
	return fmt.Sprintf("article:look:%d", id)
}

/*
*
showAdd：详情页这次要加上去的浏览数
flush：攒够10次后，一次性写入数据库的浏览数
err：错误信息
*/
func AddLook(articleID uint) (showAdd int, flush int, err error) {
	if global.Redis == nil {
		return 0, 0, errors.New("redis未连接")
	}
	n, err := global.Redis.Incr(lookKey(articleID)).Result()
	if err != nil {
		return 0, 0, err
	}

	if n%int64(lookFlushStep) == 0 {
		return lookFlushStep, lookFlushStep, nil
	}
	return int(n % int64(lookFlushStep)), 0, nil
}

func UnflushedLook(articleID uint) int {
	if global.Redis == nil {
		return 0
	}
	val, err := global.Redis.Get(lookKey(articleID)).Result()
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(val, 10, 64) // 将字符串转为数字，转为10进制，结果按 64 位整数范围检查，返回值类型是int64
	return int(n % int64(lookFlushStep))
}

func ClearLook(articleID uint) {
	if global.Redis == nil {
		return
	}
	_, err := global.Redis.Del(lookKey(articleID)).Result()
	if err != nil {
		return
	}
}
