package models

import "time"

// 个人搜索历史
type SearchHistoryModel struct {
	Model
	UserID     uint      `gorm:"index:idx_search_history_user,priority:1" json:"userID"`
	UserModel  UserModel `gorm:"foreignKey:UserID" json:"-"`
	Keyword    string    `gorm:"size:64" json:"keyword"`
	SearchedAt time.Time `gorm:"index:idx_search_history_user,priority:2" json:"searchedAt"`
}
