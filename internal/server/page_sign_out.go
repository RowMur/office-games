package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func signOut(c echo.Context) error {
	cookie := &http.Cookie{
		Name:   "auth",
		Value:  "",
		MaxAge: -1,
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusTemporaryRedirect, "/sign-in")
}
