package redis_search

import (
	"encoding/json"
	"go-star/global"
	"go-star/models"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	hotKey       = "search:hot"
	hotExpire    = 2 * time.Minute
	hotCacheSize = 50
)

func GetHot(limit int) (list []models.SearchHotModel, ok bool) {
	if global.Redis == nil {
		return nil, false
	}
	if limit <= 0 || limit > hotCacheSize {
		limit = 10
	}
	val, err := global.Redis.Get(hotKey).Result()
	if err != nil {
		return nil, false
	}
	var all []models.SearchHotModel
	// json.Unmarshal：将字符串解析成对象，然后放进all中
	if err := json.Unmarshal([]byte(val), &all); err != nil {
		return nil, false
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, true
}

func SetHot(list []models.SearchHotModel) {
	if global.Redis == nil {
		return
	}
	if len(list) > hotCacheSize {
		list = list[:hotCacheSize]
	}
	data, err := json.Marshal(list)
	if err != nil {
		return
	}
	if err := global.Redis.Set(hotKey, data, hotExpire).Err(); err != nil {
		logrus.Errorf("写入热搜缓存失败: %v", err)
	}
}

func ClearHot() {
	if global.Redis == nil {
		return
	}
	_ = global.Redis.Del(hotKey).Err()
}

func HotCacheSize() int {
	return hotCacheSize
}
