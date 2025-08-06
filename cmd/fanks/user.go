package main

import (
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"slices"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/oliverisaac/fanks/types"
	"github.com/oliverisaac/fanks/views"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func getUserByID(db *gorm.DB, id uint) (types.User, error) {
	var user types.User
	err := db.Preload("PushSubscriptions").First(&user, "id = ?", id).Error

	return user, errors.Wrap(err, "Finding user")
}

func userExists(email string, db *gorm.DB) bool {
	var user types.User
	err := db.First(&user, "email = ?", email).Error

	return err != gorm.ErrRecordNotFound
}

func signUp() echo.HandlerFunc {
	return func(c echo.Context) error {
		return render(c, 200, views.SignUpForm(nil))
	}
}

func signUpWithEmailAndPassword(db *gorm.DB, cfg types.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.FormValue("name")
		email := c.FormValue("email")
		password := c.FormValue("password")

		parsedEmail, err := mail.ParseAddress(email)
		if err != nil {
			return render(c, 422, views.SignUpForm(fmt.Errorf("Oops! That email address appears to be invalid")))
		}
		email = parsedEmail.Address

		if len(cfg.AllowSignupEmails) > 0 && !slices.Contains(cfg.AllowSignupEmails, email) {
			return render(c, 422, views.SignUpForm(fmt.Errorf("Oops! That email address is banned")))
		}

		if userExists(email, db) {
			return render(c, 422, views.SignUpForm(fmt.Errorf("Oops! It appears you are already registered")))
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			log.Fatal("Could not hash sign up password")
		}

		// Check if this is the first user
		var count int64
		if err := db.Model(&types.User{}).Count(&count).Error; err != nil {
			err := errors.Wrap(err, "Internal server error")
			return render(c, 422, views.SignUpForm(err))
		}

		role := "user"
		if count == 0 {
			role = "admin"
		}

		user := types.User{
			Name:      name,
			Email:     email,
			Password:  string(hash),
			Role:      role,
			CreatedAt: time.Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			err := errors.Wrap(err, "Create user error")
			return render(c, 422, views.SignUpForm(err))
		}

		return render(c, 200, views.SignUpForm(nil))
	}
}

func signIn(cfg types.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		return render(c, 200, views.SignInForm(cfg, nil))
	}
}

func signInWithEmailAndPassword(db *gorm.DB, cfg types.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		email := c.FormValue("email")
		password := c.FormValue("password")

		_, err := mail.ParseAddress(email)
		if err != nil {
			return render(c, 422, views.SignInForm(cfg, fmt.Errorf("Invalid email")))
		}

		var user types.User
		db.First(&user, "email = ?", email)
		if compareErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); compareErr != nil {
			return render(c, 422, views.SignInForm(cfg, fmt.Errorf("Invalid email or password")))
		}

		sess, _ := session.Get(SessionKey, c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   3600 * 24 * 365,
			HttpOnly: true,
		}

		sess.Values[SessionUserIDKey] = user.ID

		err = sess.Save(c.Request(), c.Response())
		if err != nil {
			return render(c, 422, views.SignInForm(cfg, errors.Wrap(err, "Internal server error")))
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

		return c.Redirect(http.StatusFound, "/")
	}
}
