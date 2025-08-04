package main

import (
	errs "errors"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type HomePageData struct {
	User  *User
	Notes []Note
	Err   error
}

func (d *HomePageData) WithError(err error) *HomePageData {
	d.Err = errs.Join(d.Err, err)
	return d
}

func (d *HomePageData) WithUser(u User) *HomePageData {
	d.User = &u
	return d
}

func (d *HomePageData) WithNotes(notes []Note) *HomePageData {
	d.Notes = append(d.Notes, notes...)
	return d
}

func homePageHandler(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		pageData := HomePageData{}

		if user, ok := GetSessionUser(c); ok {
			logrus.Infof("Generating homepage for user %s", user.Email)
			notes, err := user.GetNotes(db)
			if err != nil {
				pageData.WithError(err)
			}

			pageData.
				WithUser(user).
				WithNotes(notes)
		} else {
			logrus.Infof("Generating anonymous homepage")
		}

		return c.Render(200, "index", pageData)
	}
}
