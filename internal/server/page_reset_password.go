package server

import (
	"github.com/RowMur/office-games/internal/db"
	t "github.com/RowMur/office-games/internal/token"
	"github.com/RowMur/office-games/internal/views"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func resetPasswordPage(c echo.Context) error {
	token := tokenFromContext(c)
	pageContent := views.ResetPasswordPage(token.String)
	return render(c, 200, views.Page(pageContent, nil))
}

func resetPasswordFormHandler(c echo.Context) error {
	token := tokenFromContext(c)

	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm")
	if password == "" || confirmPassword == "" {
		data := views.FormData{}
		errs := views.FormErrors{}
		if password == "" {
			errs["password"] = "Password is required"
		}
		if confirmPassword == "" {
			errs["confirm"] = "Confirm password is required"
		}
		return render(c, 200, views.ResetPasswordForm(data, errs, token.String))
	}

	if password != confirmPassword {
		data := views.FormData{}
		errs := views.FormErrors{"confirm": "Passwords do not match"}
		return render(c, 200, views.ResetPasswordForm(data, errs, token.String))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		data := views.FormData{}
		errs := views.FormErrors{"submit": "Failed to reset password"}
		return render(c, 200, views.ResetPasswordForm(data, errs, token.String))
	}

	d := db.GetDB()
	user := &db.User{}
	err = d.Model(user).Where("id = ?", token.UserId).Update("password", string(hashedPassword)).Error
	if err != nil {
		data := views.FormData{}
		errs := views.FormErrors{"submit": "Failed to reset password"}
		return render(c, 200, views.ResetPasswordForm(data, errs, token.String))
	}

	return render(c, 200, views.ResetPasswordSuccess())
}

type contextWithToken struct {
	echo.Context
	token *t.Token
}

func tokenFromContext(c echo.Context) *t.Token {
	cc, ok := c.(*contextWithToken)
	if !ok {
		return nil
	}
	return cc.token
}

func resetPasswordTokenMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tokenString := c.QueryParam("token")
		if tokenString == "" {
			return c.String(400, "Token is required")
		}

		token, err := t.ParseToken(tokenString)
		if err != nil {
			return c.String(400, "Invalid token")
		}
		if token.HasExpired {
			return c.String(400, "Link has expired. Please request a new one.")
		}

		cc := &contextWithToken{c, token}
		return next(cc)
	}
}
