package models

import "time"

type TagModel struct {
	Model
	Name         string `gorm:"size:32;uniqueIndex" json:"name"` // 标签名，添加唯一索引防止同名
	ArticleCount int    `json:"articleCount"`                    // 被多少篇文章使用
}

type ArticleTagModel struct {
	ArticleID    uint         `gorm:"uniqueIndex:idx_article_tag" json:"articleID"`
	TagID        uint         `gorm:"uniqueIndex:idx_article_tag" json:"tagID"`
	ArticleModel ArticleModel `gorm:"foreignKey:ArticleID" json:"-"`
	TagModel     TagModel     `gorm:"foreignKey:TagID" json:"tag"`
	CreatedAt    time.Time    `json:"createdAt"`
}
