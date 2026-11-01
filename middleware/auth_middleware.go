package middleware

import (
	"go-star/common/res"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 只校验登录身份，不校验具体权限点
func AuthMiddleware(c *gin.Context) {
	permMiddleware("", c)
}

// PermMiddleware 返回带权限点校验的中间件
// 要求当前角色持有该权限点，否则返回权限不足
func PermMiddleware(p enum.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		permMiddleware(p, c)
	}
}

// permMiddleware 中间件公共实现：验签 → 改密校验 → 黑名单校验 → 权限点校验
// 权限判断采用白名单模式，失败即拒绝，避免脏数据或漏配导致越权
func permMiddleware(p enum.Permission, c *gin.Context) {
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
	if p != "" && !enum.HasPerm(claims.Role, p) {
		res.FailWithMsg("权限不足", c)
		c.Abort()
		return
	}
	c.Set("claims", claims)
}

// checkTokenVersion 校验token是否已因凭证变更（改密/改角色）而作废
// 只靠iat时间戳比对，不信任token里的role快照
func checkTokenVersion(claims *jwts.MyClaims, c *gin.Context) bool {
	// 没有 iat 的是改造前签发的旧token，一律视为无效（安全优先）
	if claims.IssuedAt == 0 {
		res.FailWithMsg("登录状态已失效，请重新登录", c)
		c.Abort()
		return false
	}
	if redis_jwt.HasPwdChangedAfter(claims.UserID, claims.IssuedAt) {
		res.FailWithMsg("密码已修改，请重新登录", c)
		c.Abort()
		return false
	}
	// 角色变更后旧access仍携带着旧role快照，必须同样作废，否则降级不生效
	if redis_jwt.HasRoleChangedAfter(claims.UserID, claims.IssuedAt) {
		res.FailWithMsg("账号权限已变更，请重新登录", c)
		c.Abort()
		return false
	}
	return true
}
