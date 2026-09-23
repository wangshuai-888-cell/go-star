package middleware

import (
	"fmt"
	"time"

	"go-star/common/res"
	"go-star/service/redis_service/redis_limit"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

// RateLimitByIP 按 IP 限流，用在登录等未登录接口
func RateLimitByIP(action string, max int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !redis_limit.Allow(action, ip, max, window) {
			res.FailWithMsg("请求过于频繁，请稍后再试", c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitByUser 按登录用户限流，用在评论等
func RateLimitByUser(action string, max int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		_claims, ok := c.Get("claims")
		if !ok {
			res.FailWithMsg("请登录", c)
			c.Abort()
			return
		}
		claims := _claims.(*jwts.MyClaims)
		identity := fmt.Sprintf("%d", claims.UserID)
		if !redis_limit.Allow(action, identity, max, window) {
			res.FailWithMsg("请求过于频繁，请稍后再试", c)
			c.Abort()
			return
		}
		c.Next()
	}
}
