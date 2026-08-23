package models

import "go-star/models/enum"

type LogModel struct {
	Model
	LogType     enum.LogType      `json:"logType"` // 日志类型 1 2 3
	Title       string            `gorm:"size:64" json:"title"`
	Content     string            `json:"content"`
	Level       enum.LogLevelType `json:"level"`  // 日志级别 1 2 3
	UserID      uint              `json:"userID"` // 用户ID
	UserModel   UserModel         `gorm:"foreignKey:UserID" json:"-"`
	IP          string            `gorm:"size:32" json:"ip"`
	Addr        string            `gorm:"size:64" json:"addr"`
	IsRead      bool              `json:"isRead"`      // 是否已读
	LoginStatus bool              `json:"loginStatus"` // 登录状态
	Username    string            `gorm:"size:32" json:"username"`
	Pwd         string            `gorm:"size:32" json:"pwd"`
	LoginType   enum.LoginType    `json:"loginType"`
	ServiceName string            `gorm:"size: 32" json:"serviceName"`
}
