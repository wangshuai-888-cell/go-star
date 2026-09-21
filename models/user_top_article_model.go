package models

import "time"

type UserTopArticleModel struct {
	UserID       uint         `gorm:"uniqueIndex:idx_name" json:"userId"`
	ArticleID    uint         `gorm:"uniqueIndex:idx_name" json:"articleId"`
	UserModel    UserModel    `gorm:"foreignKey:UserID" json:"-"`
	ArticleModel ArticleModel `gorm:"foreignKey:ArticleID" json:"article"`
	CreatedAt    time.Time    `json:"createdAt"`
}
