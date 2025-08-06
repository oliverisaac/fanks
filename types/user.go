package types

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name              string
	Email             string
	Password          string
	Role              string
	Notes             []Note
	PushSubscriptions []PushSubscription
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         *time.Time `gorm:"autoUpdateTime"`
	DeletedAt         *time.Time
}

func (u User) IsSet() bool {
	return u.Email != ""
}
