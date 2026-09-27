package models

import (
	"go-star/global"
	"go-star/models/enum"
)

// UserMessageModel 站内消息（个人收件箱）
type UserMessageModel struct {
	Model
	RevUserID  uint             `json:"revUserID"` // 接收人
	Type       enum.MessageType `json:"type"`      // 消息类型
	Title      string           `gorm:"size:64" json:"title"`
	Content    string           `gorm:"size:256" json:"content"`
	ArticleID  *uint            `json:"articleID"`  // 可选，关联文章
	FromUserID *uint            `json:"fromUserID"` // 可选，触发者（如关注人）
	IsRead     bool             `json:"isRead"`     // 是否已读
}

func CreateUserMessage(msg UserMessageModel) {
	_ = global.DB.Create(&msg).Error // 发消息失败不影响主流程
}
