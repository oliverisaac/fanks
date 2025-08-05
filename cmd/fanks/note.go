package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/oliverisaac/fanks/views"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func newNoteForUser(content string, user types.User) types.Note {
	return types.Note{
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
			err = errors.Wrap(err, "Saving note to db")
			logrus.Error(err)
			return render(c, 500, views.CreateNoteForm(note, err))
		}

		return render(c, 500, views.CreateNoteForm(note, nil))
	}
}
