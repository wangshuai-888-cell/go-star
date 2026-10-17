package redis_article

import (
	"encoding/json"
	"fmt"
	"go-star/global"
	"go-star/models"
	"time"

	"golang.org/x/sync/singleflight"
)

const detailExpire = 10 * time.Minute // 文章详情缓存10分钟

// sf 用来合并同一篇文章的并发查库请求（防缓存击穿）
var sf singleflight.Group

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

/*
*
获取文章详情、
缓存未命中时用 singleflight 合并并发请求：同一篇文章同时只打一条 SQL
注意：Redis 不可用时不能直接返回 false（那会导致"文章不存在"的误报），
要降级为直接查库，保证业务可用
*/
func GetDetail(id uint, article *models.ArticleModel) (ok bool) {
	// Redis可用时先查缓存
	if global.Redis != nil {
		val, err := global.Redis.Get(detailKey(id)).Result()
		if err == nil {
			// json.Unmarshal：把字符串转为结构体
			if json.Unmarshal([]byte(val), article) == nil {
				return true
			}
		}
	}

	// 未命中（或redis挂了）合并并发查库
	v, err, _ := sf.Do(detailKey(id), func() (interface{}, error) {
		var a models.ArticleModel
		if err := global.DB.Take(&a, id).Error; err != nil {
			return nil, err // 查不到也会作为结果共享，其他并发请求一起收到
		}
		SetDetail(a) // 查库成功立刻回填缓存，后面的请求直接命中
		return a, nil
	})
	if err != nil {
		return false
	}
	*article = v.(models.ArticleModel)
	return true
}

func ClearDetail(id uint) {
	if global.Redis == nil {
		return
	}
	_ = global.Redis.Del(detailKey(id)).Err()
}
