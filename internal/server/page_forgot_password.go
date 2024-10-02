package server

import (
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
)

func forgotPasswordPage(c echo.Context) error {
	pageContent := views.ForgotPasswordPage()
	return render(c, http.StatusOK, views.Page(pageContent, nil))
}

func forgotPasswordFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	if username == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "Username is required"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	d := db.GetDB()
	user := &db.User{}
	err := d.Where("username = ?", username).First(user).Error
	if err != nil {
		if db.IsRecordNotFoundError(err) {
			data := views.FormData{"username": username}
			errs := views.FormErrors{"username": "User not found"}
			return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if user.Email == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User does not have an address to send recovery email to"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	err = sendForgotPasswordEmail(user)
	if err != nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"submit": "Failed to send recovery email"}
		return render(c, http.StatusOK, views.ForgotPasswordForm(data, errs))
	}

	return render(c, http.StatusOK, views.ForgotPasswordEmailSent())
}
