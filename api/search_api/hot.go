package search_api

import (
	"context"
	"errors"
	"fmt"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/service/redis_service/redis_search"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SearchHotRequest struct {
	Limit int `form:"limit"`
}

func (SearchApi) SearchHotView(c *gin.Context) {
	var cr SearchHotRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	limit := cr.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// 给查询操作设置超时时间
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	// 在函数结束时取消上下文
	defer cancel()

	// 1.先读缓存
	if list, ok := redis_search.GetHot(limit); ok {
		logrus.Info("热搜缓存命中")
		res.OKWithData(list, c)
		return
	}
	// 2.未命中：查库（多查一些， 方便缓存复用不用limit）
	var list []models.SearchHotModel
	err := global.DB.WithContext(ctx).
		Order("search_count desc, last_search_at desc").
		Limit(redis_search.HotCacheSize()).
		Find(&list).Error
	if err != nil {
		err = fmt.Errorf("查询热搜榜：%w", err)
		if errors.Is(err, context.DeadlineExceeded) {
			logrus.Errorf("%v", err)
			res.FailWithMsg("服务繁忙，请稍后重试", c)
			return
		}
		logrus.Errorf("%v", err)
		res.FailWithMsg("获取热搜失败", c)
		return
	}

	redis_search.SetHot(list)

	if len(list) > limit {
		list = list[:limit]
	}

	res.OKWithData(list, c)
}
