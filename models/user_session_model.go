package models

import "time"

type UserSessionModel struct {
	Model
	UserID           uint      `gorm:"index:idx_user_session_list,priority:1" json:"userID"` // priority：
	UserModel        UserModel `gorm:"foreignKey:UserID" json:"-"`                           // foreignKey：外键关联的表和字段（但是不是物理外键）
	RefreshTokenHash string    `gorm:"size:64;uniqueIndex:idx_session_refresh" json:"-"`
	UserAgent        string    `gorm:"size:256" json:"userAgent"`
	IP               string    `gorm:"size:32" json:"ip"`
	Addr             string    `gorm:"size:64" json:"addr"`
	ExpiresAt        time.Time `gorm:"index:idx_user_session_list,priority:3;index:idx_session_expires" json:"expiresAt"`
	LastUsedAt       time.Time `json:"lastUsedAt"`
	IsRevoked        bool      `gorm:"index:idx_user_session_list,priority:2" json:"isRevoked"`
}
