package models

import "time"

type UserSignModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:idx_user_date;not null;comment:用户ID" json:"userID"`
	SignDate  time.Time `gorm:"uniqueIndex:idx_user_date;type:date;not null;comment:签到日期" json:"signDate"`
	CreatedAt time.Time `json:"createdAt"`
}
