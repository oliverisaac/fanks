package main

import (
	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/oliverisaac/fanks/views"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func homePageHandler(cfg types.Config, db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		pageData := types.HomePageData{Config: cfg}

		if user, ok := GetSessionUser(c); ok {
			logrus.Infof("Generating homepage for user %s", user.Email)
			notes, err := GetAllNotes(db)
			if err != nil {
				pageData.WithError(err)
			}

			for i, note := range notes {
				note.IsUserNote = note.UserID == user.ID
				notes[i] = note
			}

			pageData = pageData.
				WithUser(user).
				WithNotes(notes)
		} else {
			logrus.Infof("Generating anonymous homepage")
		}

		prompt := c.QueryParam("prompt")
		if prompt == "" {
			prompt = randomPrompt()
		}

		pageData = pageData.WithPrompt(prompt)

		return render(c, 200, views.Index(pageData))
	}
}
