package search_api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	now := time.Now()

	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var hot models.SearchHotModel
		err := tx.Where("keyword = ?", kw).Take(&hot).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询热词 keyword=%s: %w", kw, err)
			}
			hot = models.SearchHotModel{
				Keyword:      kw,
				SearchCount:  1,
				LastSearchAt: now,
			}
			if err := tx.Create(&hot).Error; err != nil {
				return fmt.Errorf("创建热词 keyword=%s: %w", kw, err)
			}
		} else {
			if err := tx.Model(&hot).Updates(map[string]any{
				"search_count":   gorm.Expr("search_count + ?", 1),
				"last_search_at": now,
			}).Error; err != nil {
				return fmt.Errorf("更新热词 keyword=%s: %w", kw, err)
			}
		}
		if err := tx.Create(&models.SearchHistoryModel{
			UserID:     claims.UserID,
			Keyword:    kw,
			SearchedAt: now,
		}).Error; err != nil {
			return fmt.Errorf("写入搜索历史 userID=%d keyword=%s: %w", claims.UserID, kw, err)
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("记录搜索失败: %v", err)
		res.FailWithMsg("记录搜索失败", c)
		return
	}
	res.OKWithMsg("记录成功", c)
}
