package server

import (
	"net/http"
	"strings"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/views"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
)

func mePageHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusUnauthorized, "/sign-in")
	}

	mePageContent := views.MePage(*user, views.FormData{"email": user.Email, "username": user.Username}, nil)
	return render(c, http.StatusOK, views.Page(mePageContent, user))
}

func meUpdateHandler(c echo.Context) error {
	user := userFromContext(c)
	if user == nil {
		return c.Redirect(http.StatusUnauthorized, "/sign-in")
	}

	username := c.FormValue("username")
	email := c.FormValue("email")

	if username == "" || email == "" {
		data := views.FormData{"username": username, "email": email}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if email == "" {
			errs["email"] = "Email is required"
		}
		falseVar := false
		return render(c, http.StatusOK, views.UserDetails(data, errs, &falseVar))
	}

	updatedUser := &db.User{}
	err := db.GetDB().Model(updatedUser).Where("id = ?", user.ID).Updates(map[string]interface{}{"username": username, "email": email}).Error
	if err != nil {
		postgresErr, ok := err.(*pgconn.PgError)
		if !ok {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		formData := views.FormData{"username": username, "email": email}
		wasSuccessful := false

		// Check if the error is a unique constraint violation
		if postgresErr.SQLState() == "23505" {
			constaintArray := strings.Split(postgresErr.ConstraintName, "_")
			columnName := constaintArray[len(constaintArray)-1]

			if columnName == "username" {
				errs := views.FormErrors{"username": "Username is taken"}
				return render(c, http.StatusOK, views.UserDetails(formData, errs, &wasSuccessful))
			}

			if columnName == "email" {
				errs := views.FormErrors{"email": "Email is taken"}
				return render(c, http.StatusOK, views.UserDetails(formData, errs, &wasSuccessful))
			}
		}

		return c.String(http.StatusInternalServerError, err.Error())
	}

	formData := views.FormData{"email": updatedUser.Email, "username": updatedUser.Username}
	truePtr := true
	return render(c, http.StatusOK, views.UserDetails(formData, views.FormErrors{}, &truePtr))
}
