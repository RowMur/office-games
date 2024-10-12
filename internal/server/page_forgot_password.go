package server

import (
	"fmt"
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func forgotPasswordPage(c echo.Context) error {
	pageContent := views.ForgotPasswordPage()
	return render(c, http.StatusOK, views.Page(pageContent, nil))
}

func (s *Server) forgotPasswordFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	if username == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "Username is required"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if user == nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User not found"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	if user.Email == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User does not have an address to send recovery email to"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	err = sendForgotPasswordEmail(c, user)
	if err != nil {
		fmt.Println("sendForgotPasswordEmail error", err.Error())
		data := views.FormData{"username": username}
		errs := views.FormErrors{"submit": "Failed to send recovery email"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	return render(c, http.StatusOK, views.ForgotPasswordEmailSent())
}
