package models

import "time"

// UserFollowModel 关注关系：UserID 关注了 FollowUserID
// 这里的UserID和FollowUserID用了uniqueIndex:idx_follow唯一索引，就是说每组数据中，UserID和FollowUserID的组合是唯一的，不能重复。
type UserFollowModel struct {
	UserID          uint      `gorm:"uniqueIndex:idx_follow" json:"userID"`
	UserModel       UserModel `gorm:"foreignKey:UserID" json:"user"`
	FollowUserID    uint      `gorm:"uniqueIndex:idx_follow" json:"followUserID"`
	FollowUserModel UserModel `gorm:"foreignKey:FollowUserID" json:"followUser"`
	CreatedAt       time.Time `json:"createdAt"`
}
