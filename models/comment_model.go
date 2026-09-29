package models

type CommentModel struct {
	Model
	Content          string          `gorm:"size:256" json:"content"`
	UserID           uint            `json:"userID"`
	UserModel        UserModel       `gorm:"foreignKey:UserID" json:"-"`
	ArticleID        uint            `json:"articleID"`
	ArticleModel     ArticleModel    `gorm:"foreignKey:ArticleID" json:"-"`
	ParentID         *uint           `json:"parentID"` // 父评论
	ParentModel      *CommentModel   `gorm:"foreignKey:ParentID" json:"-"`
	SubCommentList   []*CommentModel `gorm:"foreignKey:ParentID" json:"subCommentList"` // 子评论列表
	RootParentID     *uint           `json:"rootParentID"`                              // 根评论
	DiggCount        uint            `json:"diggCount"`                                 // 评论点赞数
	ReplyToCommentID *uint           `json:"replyToCommentID"`                          // 实际回复的那条评论
	ReplyToUserID    *uint           `json:"replyToUserID"`                             // 被回复的用户
}
