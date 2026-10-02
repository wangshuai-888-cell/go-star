package models

import "time"

// 登录会话/设备
type UserSessionModel struct {
	Model
	UserID           uint      `json:"userID"`
	UserModel        UserModel `gorm:"foreignKey:UserID" json:"-"`
	RefreshTokenHash string    `gorm:"size:64;index" json:"-"` // refresh 的 sha256，不存明文
	UserAgent        string    `gorm:"size:256" json:"userAgent"`
	IP               string    `gorm:"size:32" json:"ip"`
	Addr             string    `gorm:"size:64" json:"addr"`
	ExpiresAt        time.Time `json:"expiresAt"`
	LastUsedAt       time.Time `json:"lastUsedAt"`
	IsRevoked        bool      `json:"isRevoked"` // 退出或被踢
}
