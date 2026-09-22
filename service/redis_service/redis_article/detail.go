package redis_article

import (
	"encoding/json"
	"fmt"
	"go-star/global"
	"go-star/models"
	"time"
)

const detailExpire = 10 * time.Minute // 文章详情缓存10分钟

func detailKey(id uint) string {
	return fmt.Sprintf("article:detail:%d", id)
}

func SetDetail(article models.ArticleModel) {
	if global.Redis == nil {
		return
	}
	data, err := json.Marshal(article) // 将结构体转为JSON字节，方便存进Redis
	if err != nil {
		return
	}
	_ = global.Redis.Set(detailKey(article.ID), data, detailExpire).Err()
}

func GetDetail(id uint, article *models.ArticleModel) (ok bool) {
	if global.Redis == nil {
		return false
	}
	val, err := global.Redis.Get(detailKey(id)).Result()
	if err != nil {
		return false
	}
	// json.Unmarshal：把json字符串转为结构体
	if err := json.Unmarshal([]byte(val), article); err != nil {
		return false
	}
	return true
}

func ClearDetail(id uint) {
	if global.Redis == nil {
		return
	}
	_ = global.Redis.Del(detailKey(id)).Err()
}
