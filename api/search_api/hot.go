package search_api

import (
	"context"
	"errors"
	"fmt"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
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

	var list []models.SearchHotModel
	// 使用WithContext方法，传入上下文，确保查询操作在超时后被取消
	err := global.DB.WithContext(ctx).
		Order("search_count desc, last_search_at desc").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		err = fmt.Errorf("查询热搜榜: %w", err)
		if errors.Is(err, context.DeadlineExceeded) {
			logrus.Errorf("%v", err)
			res.FailWithMsg("服务繁忙，请稍后重试", c)
			return
		}
		logrus.Errorf("%v", err)
		res.FailWithMsg("获取热搜失败", c)
		return
	}
	res.OKWithData(list, c)
}
