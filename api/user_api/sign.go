package user_api

import (
	"go-star/common/res"
	"go-star/service/redis_service/redis_sign"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
)

// 查询本月签到记录
func (UserApi) SignInMonthView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	userID := claims.UserID

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	count, err := redis_sign.GetMonthSignCount(userID, year, month)
	if err != nil {
		res.FailWithMsg("查询失败："+err.Error(), c)
		return
	}

	// 获取具体签到日期
	days, err := redis_sign.GetMonthSignDays(userID, year, month)
	if err != nil {
		res.FailWithMsg("查询失败："+err.Error(), c)
		return
	}

	res.OKWithData(map[string]any{
		"count": count,
		"days":  days,
	}, c)
}

// 同步签到数据到Redis
func (UserApi) SyncSignDataView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	userID := claims.UserID

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	err := redis_sign.SyncHistorySignToRedis(userID, year, month)
	if err != nil {
		res.FailWithMsg("同步失败："+err.Error(), c)
		return
	}
	res.OKWithMsg("同步成功", c)
}

// 今日签到接口
func (UserApi) SignInView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	userID := claims.UserID

	now := time.Now()
	success, err := redis_sign.SignIn(userID, now)
	if err != nil {
		res.FailWithMsg("签到失败："+err.Error(), c)
		return
	}

	if !success {
		res.FailWithMsg("今日已签到", c)
		return
	}

	res.OKWithMsg("签到成功", c)
}

// 查询今日是否签到
func (UserApi) SignInTodayView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	userID := claims.UserID

	now := time.Now()
	signed, err := redis_sign.IsSigned(userID, now)
	if err != nil {
		res.FailWithMsg("查询失败: "+err.Error(), c)
		return
	}

	res.OKWithData(map[string]any{
		"signed": signed,
	}, c)
}

// 查询连续签到天数
func (UserApi) ContinuousSignInView(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)
	userID := claims.UserID

	days, err := redis_sign.GetContinuousSignDays(userID)
	if err != nil {
		res.FailWithMsg("查询失败："+err.Error(), c)
		return
	}
	res.OKWithData(map[string]any{
		"days": days,
	}, c)
}
