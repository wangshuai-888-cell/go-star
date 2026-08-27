package models

import (
	"time"
)

type Model struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type IDRequest struct {
	ID uint `json:"id" form:"id" uri:"id"`
}

type RemoveRequest struct {
	IDList []uint `json:"idList"`
}
