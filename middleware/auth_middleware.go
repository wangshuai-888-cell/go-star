package middleware

import (
	"go-star/common/res"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
	claims, err := jwts.ParseTokenByGin(c)
	if err != nil {
		res.FailWithError(err, c)
		c.Abort()
		return
	}
	if !checkTokenVersion(claims, c) {
		return
	}
	blcType, ok := redis_jwt.HasTokenBlackByGin(c)
	if ok {
		res.FailWithMsg(blcType.Msg(), c)
		c.Abort()
		return
	}
	c.Set("claims", claims)
}

func AdminMiddleware(c *gin.Context) {
	claims, err := jwts.ParseTokenByGin(c)
	if err != nil {
		res.FailWithError(err, c)
		c.Abort()
		return
	}
	if !checkTokenVersion(claims, c) { // 新增
		return
	}
	if claims.Role != enum.AdminRole {
		res.FailWithMsg("权限错误", c)
		c.Abort()
		return
	}
	blcType, ok := redis_jwt.HasTokenBlackByGin(c)
	if ok {
		res.FailWithMsg(blcType.Msg(), c)
		c.Abort()
		return
	}
	c.Set("claims", claims)
}

// checkTokenVersion token签发时间早于最近一次改密码时间 -> 该token已作废
func checkTokenVersion(claims *jwts.MyClaims, c *gin.Context) bool {
	// 没有 iat 的是改造前签发的旧token，一律视为无效（安全优先）
	if claims.IssuedAt == 0 || redis_jwt.HasPwdChangedAfter(claims.UserID, claims.IssuedAt) {
		res.FailWithMsg("密码已修改，请重新登录", c)
		c.Abort()
		return false
	}
	return true
}
