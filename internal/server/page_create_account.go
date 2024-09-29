package server

import (
	"net/http"
	"strings"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func createAccountPageHandler(c echo.Context) error {
	createAccountPageContent := views.CreateAccountPage()
	return render(c, http.StatusOK, views.Page(createAccountPageContent, nil))
}

func createAccountFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirm := c.FormValue("confirm")
	if username == "" || email == "" || password == "" || confirm == "" {
		data := views.FormData{"username": username, "email": email, "password": password, "confirm": confirm}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if email == "" {
			errs["email"] = "Email is required"
		}
		if password == "" {
			errs["password"] = "Password is required"
		}
		if confirm == "" {
			errs["confirm"] = "Confirm password is required"
		}
		return render(c, http.StatusOK, views.CreateAccountForm(data, errs))
	}

	if password != confirm {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"confirm": "Passwords do not match"}
		return render(c, http.StatusOK, views.CreateAccountForm(data, errs))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	user := &db.User{Username: username, Email: email, Password: string(hashedPassword)}
	err = db.GetDB().Create(user).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		formData := views.FormData{"username": username, "email": email}

		// Check if the error is a unique constraint violation
		if postgresErr.SQLState() == "23505" {
			constaintArray := strings.Split(postgresErr.ConstraintName, "_")
			columnName := constaintArray[len(constaintArray)-1]

			if columnName == "username" {
				errs := views.FormErrors{"username": "Username is taken"}
				return render(c, http.StatusOK, views.CreateAccountForm(formData, errs))
			}

			if columnName == "email" {
				errs := views.FormErrors{"email": "Email is taken"}
				return render(c, http.StatusOK, views.CreateAccountForm(formData, errs))
			}
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	token, err := generateToken(int(user.ID))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cookie := &http.Cookie{
		Name:  "auth",
		Value: token,
	}
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/")
	return c.NoContent(http.StatusOK)
}
