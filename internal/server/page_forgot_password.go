package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func forgotPasswordPage(c echo.Context) error {
	return render(c, http.StatusOK, views.ForgotPasswordPage())
}

func (s *Server) forgotPasswordFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	if username == "" {
		data := views.ForgotPasswordFormData{Username: username}
		errs := views.ForgotPasswordFormErrors{Username: "Username is required"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	user, err := s.db.GetUserByUsername(username)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if user == nil {
		data := views.ForgotPasswordFormData{Username: username}
		errs := views.ForgotPasswordFormErrors{Username: "User not found"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	if user.Email == "" {
		data := views.ForgotPasswordFormData{Username: username}
		errs := views.ForgotPasswordFormErrors{Username: "User does not have an address to send recovery email to"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	err = sendForgotPasswordEmail(c, user)
	if err != nil {
		data := views.ForgotPasswordFormData{Username: username}
		errs := views.ForgotPasswordFormErrors{Submit: "Failed to send recovery email"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	return render(c, http.StatusOK, views.ForgotPasswordEmailSent())
}
