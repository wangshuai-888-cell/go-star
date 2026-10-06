package search_api

import (
	"strings"
	"time"

	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
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
			hot = models.SearchHotModel{
				Keyword:      kw,
				SearchCount:  1,
				LastSearchAt: now,
			}
			if err := tx.Create(&hot).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&hot).Updates(map[string]any{
				"search_count":   gorm.Expr("search_count + ?", 1),
				"last_search_at": now,
			}).Error; err != nil {
				return err
			}
		}

		return tx.Create(&models.SearchHistoryModel{
			UserID:     claims.UserID,
			Keyword:    kw,
			SearchedAt: now,
		}).Error
	})
	if err != nil {
		res.FailWithMsg("记录搜索失败", c)
		return
	}
	res.OKWithMsg("记录成功", c)
}
