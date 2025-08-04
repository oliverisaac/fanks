package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

func newNoteForUser(content string, user User) Note {
	return Note{
		User:      user,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

func createNote(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if !ok {
			return fmt.Errorf("You must be logged in to create a note")
		}

		content := c.FormValue("content")

		note := newNoteForUser(content, user)

		if err := db.Create(&note).Error; err != nil {
			logrus.Error(errors.Wrap(err, "Saving note to db"))
			return c.Render(500, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		return c.Render(200, "note", note)
	}
}
