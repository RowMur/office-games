package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func createAccountPageHandler(c echo.Context) error {
	createAccountPageContent := views.CreateAccountPage()
	return render(c, http.StatusOK, views.Page(createAccountPageContent, nil))
}

func (s *Server) createAccountFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirm := c.FormValue("confirm")

	errs := s.us.CreateUser(username, email, password, confirm)
	if errs != nil {
		data := views.FormData{"username": username, "email": email}
		formErrs := views.FormErrors{
			"username": errs.Username.Error(),
			"email":    errs.Email.Error(),
			"password": errs.Password.Error(),
			"confirm":  errs.Confirm.Error(),
		}

		return render(c, http.StatusOK, views.CreateAccountForm(data, formErrs))
	}

	token, loginErrs := s.us.Login(username, password)
	if loginErrs != nil {
		data := views.FormData{"username": username, "email": email}
		formErrs := views.FormErrors{
			"username": loginErrs.Username.Error(),
			"password": loginErrs.Password.Error(),
		}

		return render(c, http.StatusOK, views.CreateAccountForm(data, formErrs))
	}

	cookie := &http.Cookie{
		Name:  "auth",
		Value: token,
	}
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/")
	return c.NoContent(http.StatusOK)
}
