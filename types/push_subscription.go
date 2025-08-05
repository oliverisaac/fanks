package types

import (
	"gorm.io/gorm"
)

type PushSubscription struct {
	gorm.Model
	UserID       uint
	Endpoint     string
	P256DH       string
	Auth         string
	Keys         string
}
