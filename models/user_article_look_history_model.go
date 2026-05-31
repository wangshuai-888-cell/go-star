package models

// 用户浏览的文章历史表
type UserArticleLookHistoryModel struct {
	Model
	UserID       uint         `json:"userId"`
	UserModel    UserModel    `gorm:"foreignKey:UserID" json:"-"`
	ArticleID    uint         `json:"articleId"`
	ArticleModel ArticleModel `gorm:"foreignKey:ArticleID" json:"-"`
}
