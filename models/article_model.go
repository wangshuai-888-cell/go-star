package models

type ArticleModel struct {
	Model
	Title        string    `gorm:"size:32" json:"title"`
	Abstract     string    `gorm:"size:256" json:"abstract"` // 摘要
	Content      string    `json:"content"`
	CategoryID   uint      `json:"categoryID"`                                   // 分类的ID
	TagList      []string  `gorm:"type:longtext;serializer:json" json:"tagList"` // 标签列表
	Cover        string    `gorm:"size:256" json:"cover"`                        // 文章的封面                        // 封面
	UserID       uint      `json:"userID"`
	UserModel    UserModel `gorm:"foreignKey:UserID" json:"-"`
	LookCount    int       `json:"lookCount"`    // 浏览数
	DiggCount    int       `json:"diggCount"`    // 点赞数
	CommentCount int       `json:"commentCount"` // 评论数
	CollectCount int       `json:"collectCount"` // 收藏数
	OpenComment  bool      `json:"openComment"`  // 开启评论
	Status       int8      `json:"status"`       // 状态(草稿、审核中、已发布)
}
