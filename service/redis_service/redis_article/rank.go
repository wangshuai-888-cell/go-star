package redis_article

import (
	"fmt"
	"go-star/global"
	"strconv"
)

const hotKey = "article:hot"

const (
	ScoreLook    = 1 // 浏览量
	ScoreDigg    = 3 // 点赞
	ScoreComment = 2 // 评论
	ScoreCollect = 2 // 收藏
)

func AddHotScore(articleID uint, score float64) {
	if global.Redis == nil || score == 0 {
		return
	}
	// 给指定ID的文件加分数
	_ = global.Redis.ZIncrBy(hotKey, score, fmt.Sprintf("%d", articleID)).Err()
}

func RemoveHot(articleID uint) {
	if global.Redis == nil {
		return
	}
	// 删掉hotKey集合中的指定文章ID
	_ = global.Redis.ZRem(hotKey, fmt.Sprintf("%d", articleID)).Err()
}

func TopHot(limit int64) ([]uint, error) {
	if global.Redis == nil {
		return nil, fmt.Errorf("redis未连接")
	}
	if limit <= 0 {
		limit = 10
	}
	// 从redis中倒序获取前十个文章ID
	vals, err := global.Redis.ZRevRange(hotKey, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(vals))
	for _, v := range vals {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	return ids, nil
}
