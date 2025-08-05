package types

import (
	"time"

	"gorm.io/gorm"
)

type Note struct {
	gorm.Model
	UserID    uint
	User      User
	Content   string
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`
	DeletedAt *time.Time
}
