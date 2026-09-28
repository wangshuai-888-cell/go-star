package data_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_article"
	"time"

	"github.com/gin-gonic/gin"
)

type OverviewResponse struct {
	UserTotal     int64                 `json:"userTotal"`
	UserToday     int64                 `json:"userToday"`
	ArticleTotal  int64                 `json:"articleTotal"`
	ArticleToday  int64                 `json:"articleToday"`
	ArticleReview int64                 `json:"articleReview"`
	CommentTotal  int64                 `json:"commentTotal"`
	CommentToday  int64                 `json:"commentToday"`
	LookTotal     int64                 `json:"lookTotal"`
	LoginToday    int64                 `json:"loginToday"`
	HotList       []models.ArticleModel `json:"hotList"`
}

func (DataApi) DataOverviewView(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var out OverviewResponse

	global.DB.Model(&models.UserModel{}).Count(&out.UserTotal)
	global.DB.Model(&models.UserModel{}).Where("created_at >= ?", today).Count(&out.UserToday)

	global.DB.Model(&models.ArticleModel{}).Where("status = ?", enum.ArticleStatusPublished).Count(&out.ArticleTotal)
	global.DB.Model(&models.ArticleModel{}).Where("status = ? AND created_at >= ?", enum.ArticleStatusPublished, today).Count(&out.ArticleToday)
	global.DB.Model(&models.ArticleModel{}).Where("status = ?", enum.ArticleStatusReview).Count(&out.ArticleReview)
	global.DB.Model(&models.CommentModel{}).Count(&out.CommentTotal)
	global.DB.Model(&models.CommentModel{}).Where("created_at >= ?", today).Count(&out.CommentToday)

	// 浏览量合计（库里面已落库的）
	// COALESCE：是一个函数，返回不为0的第一个参数。SUM：是求和。Scan：把计算结果赋值给out.LookTotal。
	global.DB.Model(&models.ArticleModel{}).Select("COALESCE(SUM(look_count), 0)").Scan(&out.LookTotal)

	// 今日登录成功（登录日志类型 = 1）
	global.DB.Model(&models.LogModel{}).Where("log_type = ? AND login_status = ? AND created_at >= ?", enum.LoginLogType, true, today).
		Count(&out.LoginToday)

	// 热门 Top10
	out.HotList = []models.ArticleModel{}
	if ids, err := redis_article.TopHot(10); err == nil && len(ids) > 0 {
		var list []models.ArticleModel
		// 下方为简写，等同于global.DB.Model(&models.ArticleModel{}).Where("id IN ?", ids).Find(&list)
		if err := global.DB.Where("id IN ?", ids).Find(&list).Error; err == nil {
			m := make(map[uint]models.ArticleModel, len(list))
			for _, a := range list {
				m[a.ID] = a
			}
			for _, id := range ids {
				if a, ok := m[id]; ok {
					a.LookCount += redis_article.UnflushedLook(a.ID) // 把还未写入数据库的浏览数也加上去
					out.HotList = append(out.HotList, a)
				}
			}
		}

	}
	res.OKWithData(out, c)
}
