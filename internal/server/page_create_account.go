package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func createAccountPageHandler(c echo.Context) error {
	return render(c, http.StatusOK, views.CreateAccountPage())
}

func (s *Server) createAccountFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirm := c.FormValue("confirm")

	errs := s.us.CreateUser(username, email, password, confirm)
	if errs != nil {
		data := views.CreateAccountFormData{Username: username, Email: email}
		formErrs := views.CreateAccountFormErrors{
			Username: errs.Username,
			Email:    errs.Email,
			Password: errs.Password,
			Confirm:  errs.Confirm,
		}

		return render(c, http.StatusOK, views.CreateAccountForm(data, formErrs))
	}

	token, loginErrs := s.us.Login(username, password)
	if loginErrs != nil {
		data := views.CreateAccountFormData{Username: username, Email: email}
		formErrs := views.CreateAccountFormErrors{
			Username: loginErrs.Username,
			Password: loginErrs.Password,
		}

		return render(c, http.StatusOK, views.CreateAccountForm(data, formErrs))
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
