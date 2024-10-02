package server

import (
	"errors"
	"net/http"

	"github.com/RowMur/office-games/internal/db"
	"github.com/RowMur/office-games/internal/token"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func signInHandler(c echo.Context) error {
	signInPageContent := views.SignInPage()
	return render(c, http.StatusOK, views.Page(signInPageContent, nil))
}

func signInFormHandler(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	if username == "" || password == "" {
		data := views.FormData{"username": username}
		errs := views.FormErrors{}
		if username == "" {
			errs["username"] = "Username is required"
		}
		if password == "" {
			errs["password"] = "Password is required"
		}
		return render(c, http.StatusOK, views.SignInForm(data, errs))
	}

	user := &db.User{}
	err := db.GetDB().Where("username = ?", username).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"username": "User not found"}
		return render(c, http.StatusOK, views.SignInForm(data, errs))
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		data := views.FormData{"username": username}
		errs := views.FormErrors{"password": "Invalid password"}
		return render(c, http.StatusOK, views.SignInForm(data, errs))
	}

	token, err := token.GenerateToken(user.ID, token.AuthenticationToken)
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
