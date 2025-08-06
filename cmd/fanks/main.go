package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/oliverisaac/fanks/static"
	"github.com/oliverisaac/fanks/types"
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

const SessionKey = "session"
const UserKey = "session-user"
const SessionUserIDKey = "userid"

func render(ctx echo.Context, status int, t templ.Component) error {
	ctx.Response().Writer.WriteHeader(status)

	err := t.Render(ctx.Request().Context(), ctx.Response().Writer)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
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

	tz := os.Getenv("TZ")
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return errors.Wrap(err, "failed to load timezone")
		}
		time.Local = loc
	}

	cfg, err := types.ConfigFromEnv()
	if err != nil {
		return errors.Wrap(err, "Loading config from env")
	}

	e := echo.New()

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
			logrus.Error(errors.Wrap(err, "recovered panic:"))
			for _, l := range strings.Split(string(stack), "\n") {
				logrus.Errorf("stack: %s", strings.ReplaceAll(l, "\t", "  "))
			}
			return nil
		},
		DisableErrorHandler: false,
	}))

	e.Use(middleware.Secure())

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/healthz"
		},
	}))

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to connect database")
	}

	err = db.AutoMigrate(&types.User{}, &types.Note{}, &types.PushSubscription{})
	if err != nil {
		return errors.Wrap(err, "Failed to migrate")
	}

	err = startNotificationWorker(cfg, db)
	if err != nil {
		return errors.Wrap(err, "Failed to setup notifciation worker")
	}

	store := sessions.NewCookieStore(cfg.CookeSecret)
	e.Use(session.Middleware(store))
	e.Use(UserMiddleware(db))

	e.GET("/serviceWorker.js", func(c echo.Context) error {
		sw, err := static.FS.ReadFile("serviceWorker.js")
		if err != nil {
			return errors.Wrap(err, "reading service worker from embed fs")
		}
		return c.Blob(http.StatusOK, "application/javascript", sw)
	})

	// Pages
	e.GET("/", homePageHandler(cfg, db))
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Blocks
	e.GET("/auth/sign-in", signIn(cfg))
	e.POST("/auth/sign-in", signInWithEmailAndPassword(db, cfg))
	if cfg.AllowSignup || len(cfg.AllowSignupEmails) > 0 {
		e.GET("/auth/sign-up", signUp())
		e.POST("/auth/sign-up", signUpWithEmailAndPassword(db, cfg))
	}
	e.POST("/auth/sign-out", signOut())

	// notes
	e.GET("/note/create", createNoteNoPrompt(db))
	e.POST("/note/create", createNote(db))
	e.DELETE("/note/:id", deleteNote(db))

	// push
	e.POST("/push/subscribe", saveSubscription(db))
	e.POST("/push/unsubscribe", removeSubscription(db))
	e.POST("/push/trigger", triggerPushes())

	return e.Start(":8080")
}

func UserMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, _ := session.Get(SessionKey, c)
			if sess.Values[SessionUserIDKey] != nil {
				userID := sess.Values[SessionUserIDKey].(uint)
				user, err := getUserByID(db, userID)
				if err != nil {
					return errors.Wrap(err, "getting user by id")
				}
				c.Set(UserKey, user)
			}
			return next(c)
		}
	}
}

func GetSessionUser(c echo.Context) (types.User, bool) {
	u := c.Get(UserKey)
	if u != nil {
		user := u.(types.User)
		logrus.Debugf("Found session user %s", user.Email)
		return user, true
	}
	return types.User{}, false
}
