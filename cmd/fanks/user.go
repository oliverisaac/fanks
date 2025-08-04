package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"slices"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type FormData struct {
	Errors map[string]string
	Values map[string]string
}

func newFormData() FormData {
	return FormData{
		Errors: map[string]string{},
		Values: map[string]string{},
	}
}

func userExists(email string, db *gorm.DB) bool {
	var user User
	err := db.First(&user, "email = ?", email).Error

	return err != gorm.ErrRecordNotFound
}

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

func newUser(name string, email string, password string, role string, created_at time.Time, updated_at *time.Time) User {
	return User{
		Name:      name,
		Email:     email,
		Password:  password,
		Role:      role,
		CreatedAt: created_at,
		UpdatedAt: updated_at,
	}
}

func (u User) IsSet() bool {
	return u.Email != ""
}

func (u User) GetNotes(db *gorm.DB) ([]Note, error) {
	ret := []Note{}
	result := db.Preload("User").Where("user_id = ?", u.ID).Find(&ret)
	if result.Error != nil {
		return nil, errors.Wrapf(result.Error, "Looking for notes owned by user %q", u.Email)
	}
	return ret, nil
}

func signUp() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(200, "sign-up-form", nil)
	}
}

func signUpWithEmailAndPassword(cfg Config, db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")
		password := c.FormValue("password")

		parsedEmail, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}
		email = parsedEmail.Address

		if len(cfg.AllowSignupEmails) > 0 && !slices.Contains(cfg.AllowSignupEmails, email) {
			return c.Render(422, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address is banned",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		if userExists(email, db) {
			return c.Render(422, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears you are already registered",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			log.Fatal("Could not hash sign up password")
		}

		// Check if this is the first user
		var count int64
		if err := db.Model(&User{}).Count(&count).Error; err != nil {
			return c.Render(500, "sign-up-form", FormData{
				Errors: map[string]string{
					"general": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		role := "user"
		if count == 0 {
			role = "admin"
		}

		user := User{
			Name:      name,
			Email:     email,
			Password:  string(hash),
			Role:      role,
			CreatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			return c.Render(500, "sign-up-form", FormData{
				Errors: map[string]string{
					"email": "Oops! It appears we have had an error",
				},
				Values: map[string]string{},
			})
		}

		return c.Render(200, "index", nil)
	}
}

func signIn() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(200, "sign-in-form", nil)
	}
}

func signInWithEmailAndPassword(db *gorm.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		password := c.FormValue("password")

		_, err := mail.ParseAddress(email)
		if err != nil {
			return c.Render(422, "sign-in-form", FormData{
				Errors: map[string]string{
					"email": "Oops! That email address appears to be invalid",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		var user User
		db.First(&user, "email = ?", email)
		if compareErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); compareErr != nil {
			return c.Render(422, "sign-in-form", FormData{
				Errors: map[string]string{
					"email": "Oops! Email address or password is incorrect.",
				},
				Values: map[string]string{
					"email": email,
				},
			})
		}

		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   3600 * 24 * 365,
			HttpOnly: true,
		}

		userBytes, err := json.Marshal(user)
		if err != nil {
			fmt.Println("error marshalling user value")
			return err
		}

		sess.Values["user"] = userBytes

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			fmt.Println("error saving session: ", err)
			return err
		}

		return c.Redirect(http.StatusFound, "/")
	}
}

func signOut() echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get("session", c)
		sess.Options.MaxAge = -1
		err := sess.Save(c.Request(), c.Response())
		if err != nil {
			fmt.Println("error saving session")
			return err
		}

		return c.Render(200, "index", nil)
	}
}
