package models

// 已消费的搜索事件：event_id 唯一，用来做幂等
type SearchEventModel struct {
	Model
	EventID string `gorm:"size:36;uniqueIndex:idx_search_event_id" json:"eventID"`
	UserID  uint   `json:"userID"`
	Keyword string `gorm:"size:64" json:"keyword"`
}
