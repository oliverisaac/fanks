package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/oliverisaac/goli"
	"github.com/oliverisaac/fanks/static"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	goli.InitLogrus(logrus.DebugLevel)
}

const UserKey = "session-user"

type Template struct {
	tmpl *template.Template
}

func newTemplate() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("template/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("error loading godotenv")
	}

	e := echo.New()

	e.Renderer = newTemplate()

	e.StaticFS("/static", static.FS)

	e.Use(middleware.Recover())

	e.Use(middleware.Secure())

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	store := sessions.NewCookieStore([]byte(os.Getenv("THANX_COOKIE_STORE_SECRET")))
	e.Use(session.Middleware(store))
	e.Use(UserMiddleware())

	db, err := gorm.Open(sqlite.Open(os.Getenv("THANX_DB_PATH")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	gormTables := []any{
		User{},
		Note{},
	}
	for _, t := range gormTables {
		err = db.AutoMigrate(&t)
		if err != nil {
			logrus.Fatal(errors.Wrap(err, "Failed to migrate"))
		}
	}

	// Pages
	e.GET("/", homePageHandler(db))

	// Blocks
	e.GET("/auth/sign-in", signIn())
	e.POST("/auth/sign-in", signInWithEmailAndPassword(db))
	e.GET("/auth/sign-up", signUp())
	e.POST("/auth/sign-up", signUpWithEmailAndPassword(db))
	e.POST("/auth/sign-out", signOut())

	// notes
	e.POST("/note/create", createNote(db))

	e.Logger.Fatal(e.Start(":8080"))
}

func UserMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get("session", c)
			if sess.Values["user"] != nil {
				var user User
				err := json.Unmarshal(sess.Values["user"].([]byte), &user)
				if err != nil {
					fmt.Println("error unmarshalling user value")
					return err
				}
				c.Set(UserKey, user)
			}
			return next(c)
		}
	}
}

func GetSessionUser(c echo.Context) (User, bool) {
	u := c.Get(UserKey)
	if u != nil {
		user := u.(User)
		logrus.Debugf("Found session user %s", user.Email)
		return user, true
	}
	return User{}, false
}
