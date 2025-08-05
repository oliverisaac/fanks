package types

import (
	"time"

	"gorm.io/gorm"
)

type Note struct {
	gorm.Model
	UserID     uint
	IsUserNote bool `gorm:-`
	User       User
	Content    string
	Prompt     string     `gorm:"default:'Today I am grateful for...'"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  *time.Time `gorm:"autoUpdateTime"`
	DeletedAt  *time.Time
}
