package types

import (
	errs "errors"
)

type HomePageData struct {
	User   *User
	Config Config
	Notes  []Note
	Err    error
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
