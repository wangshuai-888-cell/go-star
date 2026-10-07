package search_api

import (
	"strings"
	"time"

	"go-star/common/res"
	"go-star/service/kafka_service"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SearchLogRequest struct {
	Keyword string `json:"keyword" binding:"required,max=64"`
}

func normalizeKeyword(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (SearchApi) SearchLogView(c *gin.Context) {
	var cr SearchLogRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}
	kw := normalizeKeyword(cr.Keyword)
	if kw == "" {
		res.FailWithMsg("关键词不能为空", c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	err := kafka_service.ProduceSearch(kafka_service.SearchEvent{
		UserID:     claims.UserID,
		Keyword:    kw,
		SearchedAt: time.Now(),
	})
	if err != nil {
		logrus.Errorf("生产搜索事件失败: %v", err)
		res.FailWithMsg("记录失败", c)
		return
	}
	res.OKWithMsg("记录成功", c)
}
