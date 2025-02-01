package server

import (
	"fmt"
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func signInHandler(c echo.Context) error {
	fmt.Printf("%+v\n", c.QueryParams())
	fromParam := c.QueryParam("from")
	if fromParam != "" {
		cookie := &http.Cookie{
			Name:  "from",
			Value: fromParam,
		}
		c.SetCookie(cookie)
	}
	return render(c, http.StatusOK, views.SignInPage())
}

func (s *Server) signInFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	token, errs := s.us.Login(username, password)
	if errs != nil {
		if errs.Error != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, errs.Error.Error())
		}
		data := views.SignInFormData{Username: username}
		formErrors := views.SignInFormErrors{
			Username: errs.Username,
			Password: errs.Password,
		}
		return render(c, http.StatusOK, views.SignInForm(data, formErrors))
	}

	fromCookie, err := c.Cookie("from")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if fromCookie != nil {
		c.Response().Header().Set("HX-Redirect", fromCookie.Value)
		return c.NoContent(http.StatusOK)
	}

	cookie := &http.Cookie{
		Name:  "auth",
		Value: token,
	}
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/")
	return c.NoContent(http.StatusOK)
}
