package user_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/utils/hash"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
)

// 列表返回给前端的结构（不暴露 refresh hash）
type SessionItem struct {
	ID         uint      `json:"id"`
	UserAgent  string    `json:"userAgent"`
	IP         string    `json:"ip"`
	Addr       string    `json:"addr"`
	ExpiresAt  time.Time `json:"expiresAt"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	IsCurrent  bool      `json:"isCurrent"` // 是否当前这台设备
	CreatedAt  time.Time `json:"createdAt"`
}

type SessionListRequest struct {
	RefreshToken string `form:"refreshToken" binding:"required"` // 用来标记当前设备
}

// 登录设备列表
func (UserApi) SessionListView(c *gin.Context) {
	var cr SessionListRequest
	if err := c.ShouldBindQuery(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var list []models.UserSessionModel
	err := global.DB.Where(
		"user_id = ? AND is_revoked = ? AND expires_at > ?",
		claims.UserID, false, time.Now(),
	).Order("last_used_at desc").Find(&list).Error
	if err != nil {
		res.FailWithMsg("获取设备列表失败", c)
		return
	}

	currentHash := hash.SHA256Hex(cr.RefreshToken)
	items := make([]SessionItem, 0, len(list))
	for _, s := range list {
		items = append(items, SessionItem{
			ID:         s.ID,
			UserAgent:  s.UserAgent,
			IP:         s.IP,
			Addr:       s.Addr,
			ExpiresAt:  s.ExpiresAt,
			LastUsedAt: s.LastUsedAt,
			IsCurrent:  s.RefreshTokenHash == currentHash,
			CreatedAt:  s.CreatedAt,
		})
	}

	res.OKWithData(items, c)
}

// 踢下线：作废指定会话
func (UserApi) SessionRemoveView(c *gin.Context) {
	var idCr models.IDRequest
	if err := c.ShouldBindUri(&idCr); err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	result := global.DB.Model(&models.UserSessionModel{}).
		Where("id = ? AND user_id = ? AND is_revoked = ?", idCr.ID, claims.UserID, false).
		Update("is_revoked", true)
	if result.Error != nil {
		res.FailWithMsg("操作失败", c)
		return
	}
	if result.RowsAffected == 0 {
		res.FailWithMsg("会话不存在或已失效", c)
		return
	}

	res.OKWithMsg("已踢下线", c)
}
