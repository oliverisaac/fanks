package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type DashboardPageData struct {
	User User
}

func newDashboardData(user User) DashboardPageData {
	return DashboardPageData{
		User: user,
	}
}

func dashboardPageHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := GetSessionUser(c)
		if ok {
			return c.Render(200, "dashboard", newDashboardData(user))
		}

		return c.Redirect(http.StatusFound, "/")
	}
}
