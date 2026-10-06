package models

import "time"

// 全站热词：一个词一行
type SearchHotModel struct {
	Model
	Keyword      string    `gorm:"size:64;uniqueIndex:idx_search_hot_keyword" json:"keyword"`
	SearchCount  uint      `gorm:"index:idx_search_hot_rank,priority:1" json:"searchCount"`
	LastSearchAt time.Time `gorm:"index:idx_search_hot_rank,priority:2" json:"lastSearchAt"`
}
