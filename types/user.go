package types

import (
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name      string
	Email     string
	Password  string
	Role      string
	Notes     []Note
	CreatedAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`
	DeletedAt *time.Time
}

func (u User) IsSet() bool {
	return u.Email != ""
}

func (u User) GetNotes(db *gorm.DB) ([]Note, error) {
	ret := []Note{}
	result := db.Preload("User").Where("user_id = ?", u.ID).Order("created_at DESC").Find(&ret)
	if result.Error != nil {
		return nil, errors.Wrapf(result.Error, "Looking for notes owned by user %q", u.Email)
	}
	return ret, nil
}
