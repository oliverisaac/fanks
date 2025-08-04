package main

import (
	"encoding/json"
	errs "errors"
	"fmt"
	"html/template"
	"io"
	"net/mail"
	"path"
	"strconv"
	"strings"

	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/oliverisaac/fanks/static"
	"github.com/oliverisaac/goli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	_ "github.com/ncruces/go-sqlite3/embed"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
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
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

type Config struct {
	AllowSignup       bool
	AllowSignupEmails []string
	CookeSecret       []byte
	DBPath            string
}

func main() {
	err := run()
	if err != nil {
		logrus.Fatal(err)
	}
}

func run() error {
	err := godotenv.Load(".env")
	if err != nil {
		logrus.Error(errors.Wrap(err, "Failed to load .env"))
	}

	cfg, err := configFromEnv()
	if err != nil {
		return errors.Wrap(err, "Loading config from env")
	}

	e := echo.New()

	e.Renderer = newTemplate()

	e.StaticFS("/static", static.FS)

	origErrHandler := e.HTTPErrorHandler
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		logrus.Error(err)
		origErrHandler(err, c)
	}

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		Skipper:           middleware.DefaultSkipper,
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
		LogLevel:          log.ERROR,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logrus.Error(errors.Wrap(err, "recovered panic"))
			return nil
		},
		DisableErrorHandler: false,
	}))

	e.Use(middleware.Secure())

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
	}))

	store := sessions.NewCookieStore(cfg.CookeSecret)
	e.Use(session.Middleware(store))
	e.Use(UserMiddleware())

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to connect database")
	}

	err = db.AutoMigrate(&User{}, &Note{})
	if err != nil {
		return errors.Wrap(err, "Failed to migrate")
	}

	// Pages
	e.GET("/", homePageHandler(cfg, db))

	// Blocks
	e.GET("/auth/sign-in", signIn(cfg))
	e.POST("/auth/sign-in", signInWithEmailAndPassword(db))
	if cfg.AllowSignup || len(cfg.AllowSignupEmails) > 0 {
		e.GET("/auth/sign-up", signUp())
		e.POST("/auth/sign-up", signUpWithEmailAndPassword(cfg, db))
	}
	e.POST("/auth/sign-out", signOut())

	// notes
	e.POST("/note/create", createNote(db))

	return e.Start(":8080")
}

func configFromEnv() (Config, error) {
	ret := Config{}
	var retErr error
	var err error

	ret.AllowSignup, err = strconv.ParseBool(goli.DefaultEnv("FANKS_ALLOW_SIGNUP", "false"))
	if err != nil {
		retErr = errs.Join(retErr, errors.Wrap(err, "parsing FANKS_ALLOW_SIGNUP"))
	}

	allowedEmails := strings.Split(os.Getenv("FANKS_ALLOW_SIGNUP_EMAILS"), ",")
	for _, e := range allowedEmails {
		if e == "" {
			continue
		}
		email, err := mail.ParseAddress(e)
		if err != nil {
			retErr = errs.Join(retErr, errors.Wrapf(err, "parsing email %q", e))
		} else {
			ret.AllowSignupEmails = append(ret.AllowSignupEmails, email.Address)
		}
	}
	logrus.Infof("Allowed signup emails: %v", ret.AllowSignupEmails)

	cookieSecret, ok := os.LookupEnv("FANKS_COOKIE_STORE_SECRET")
	if !ok {
		retErr = errs.Join(retErr, fmt.Errorf("You must define env FANKS_COOKIE_STORE_SECRET"))
	} else {
		ret.CookeSecret = []byte(cookieSecret)
	}

	ret.DBPath, ok = os.LookupEnv("FANKS_DB_PATH")
	if !ok {
		retErr = errs.Join(retErr, fmt.Errorf("You must define env FANKS_DB_PATH"))
	} else if _, err := os.Stat(path.Dir(ret.DBPath)); err != nil {
		retErr = errs.Join(retErr, errors.Wrap(err, "Directory for FANKS_DB_PATH must exist"))
	}

	return ret, retErr
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
